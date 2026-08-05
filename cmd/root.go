package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"

	"log"
	"log/slog"
	"os"
	"runtime/pprof"
	"time"

	"github.com/gofrs/flock"
	backend "github.com/kumneger0/ytmusic-tui/backend"
	"github.com/kumneger0/ytmusic-tui/internal/config"
	"github.com/kumneger0/ytmusic-tui/internal/headless"
	logSetup "github.com/kumneger0/ytmusic-tui/internal/logger"
	"github.com/kumneger0/ytmusic-tui/internal/youtube"
	ytMusicClient "github.com/kumneger0/ytmusic-tui/internal/yt-music-client"
	"go.dalton.dog/bubbleup"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/kumneger0/ytmusic-tui/internal/mpris"
	"github.com/kumneger0/ytmusic-tui/internal/types"
	"github.com/kumneger0/ytmusic-tui/internal/ui"
)

var (
	Program *tea.Program
)

func newRootCmd(version string, debug bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ytmusic-tui",
		Short: "youtube music player",
		RunE: func(cmd *cobra.Command, args []string) error {
			legacyLockPath := filepath.Join(os.TempDir(), "clispot.lock")
			if pidBytes, err := os.ReadFile(legacyLockPath); err == nil {
				if pid, err := strconv.Atoi(string(pidBytes)); err == nil && isProcessRunning(pid) {
					showAnotherProcessIsRunning(legacyLockPath)
					os.Exit(1)
				}
				_ = os.Remove(legacyLockPath)
			}

			lockFilePath := filepath.Join(os.TempDir(), "ytmusic-tui.lock")

			fileLock := flock.New(lockFilePath)
			locked, err := fileLock.TryLock()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error trying to acquire lock: %v\n", err)
				os.Exit(1)
			}

			if !locked {
				showAnotherProcessIsRunning(lockFilePath)
				os.Exit(1)
			}
			defer func() {
				if err := fileLock.Unlock(); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not unlock file: %v\n", err)
				}
				_ = os.Remove(lockFilePath)
			}()

			if runtime.GOOS != "windows" {
				pid := os.Getpid()
				if err := os.WriteFile(lockFilePath, []byte(strconv.Itoa(pid)), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not write PID to lock file: %v\n", err)
				}
			}
			return runRoot(cmd, debug)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if memFile != "" && debug {
				f, err := os.Create(memFile)
				if err != nil {
					log.Fatal("could not create memory profile: ", err)
				}
				defer f.Close()
				if err := pprof.WriteHeapProfile(f); err != nil {
					log.Fatal("could not write memory profile: ", err)
				}
			}
			if cpuFile != "" {
				pprof.StopCPUProfile()
			}
		},
	}

	cmd.AddCommand(newVersionCmd(version))
	cmd.AddCommand(ytmusicTuiLog())
	cmd.AddCommand(ManCmd(cmd))
	cmd.AddCommand(installDeps())
	cmd.AddCommand(newLoginCmd())
	cmd.AddCommand(newExtractCookieCmd())
	return cmd
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func showAnotherProcessIsRunning(lockFilePath string) {
	if runtime.GOOS == "windows" {
		fmt.Fprintf(os.Stderr, "Another instance of ytmusic-tui is already running.\n")
		return
	}
	pidBytes, readErr := os.ReadFile(lockFilePath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return
		}
		fmt.Fprintf(os.Stderr, "Error reading lock file: %v\n", readErr)
		os.Exit(1)
	}
	pid, parseErr := strconv.Atoi(string(pidBytes))

	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "Error parsing PID from lock file: %v\n", parseErr)
		os.Exit(1)
	}

	if !isProcessRunning(pid) {
		fmt.Fprintf(os.Stderr, "Another instance of ytmusic-tui is not running (stale lock file for PID %d).\n", pid)
		fmt.Fprintf(os.Stderr, "Please try removing %s and running again if this persists.\n", lockFilePath)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Another instance of ytmusic-tui is already running (PID: %d).\n", pid)
}

func startPythonBackend(debug bool) (*exec.Cmd, error) {
	if debug {
		return nil, nil
	}
	return backend.StartBackend(backend.PythonBackend)
}

func runRoot(cmd *cobra.Command, debug bool) error {
	debugDir, err := cmd.Flags().GetString("debug-dir")
	configFromFile := config.GetUserConfig(runtime.GOOS)

	if !cmd.Flags().Changed("debug-dir") && configFromFile.DebugDir != nil {
		debugDir = *configFromFile.DebugDir
	}

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	cacheDir, err := cmd.Flags().GetString("cache-dir")
	if !cmd.Flags().Changed("cache-dir") && configFromFile.CacheDir != nil {
		cacheDir = *configFromFile.CacheDir
	}
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	isCacheDisabled, err := cmd.Flags().GetBool("disable-cache")
	if !cmd.Flags().Changed("disable-cache") {
		isCacheDisabled = configFromFile.CacheDisabled
	}
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err := os.MkdirAll(debugDir, 0755); err != nil {
		fmt.Printf("failed to create debug directory '%s': %v\n", debugDir, err)
		os.Exit(1)
	}

	fileInfo, err := os.Stat(debugDir)
	if err != nil {
		fmt.Printf("failed to stat debug directory '%s': %v\n", debugDir, err)
		os.Exit(1)
	}

	if !fileInfo.IsDir() {
		fmt.Printf("the debug path '%v' is not a directory\n", debugDir)
		os.Exit(1)
	}

	ytDlpArgs := config.YtDlpArgs{
		CookiesFromBrowser: nil,
		Cookies:            nil,
	}

	cookiesFromBrowser, err := cmd.Flags().GetString("cookies-from-browser")

	if !cmd.Flags().Changed("cookies-from-browser") {
		if configFromFile.YtDlpArgs != nil && configFromFile.YtDlpArgs.CookiesFromBrowser != nil {
			cookiesFromBrowser = *configFromFile.YtDlpArgs.CookiesFromBrowser
		}
	}

	if err != nil {
		slog.Error(err.Error())
	}

	if cookiesFromBrowser != "" {
		ytDlpArgs.CookiesFromBrowser = (&cookiesFromBrowser)
	}

	cookiesFile, err := cmd.Flags().GetString("cookies")

	if !cmd.Flags().Changed("cookies") {
		if configFromFile.YtDlpArgs != nil && configFromFile.YtDlpArgs.Cookies != nil {
			cookiesFile = *configFromFile.YtDlpArgs.Cookies
		}
	}

	if err != nil {
		slog.Error(err.Error())
	}

	if cookiesFile != "" {
		ytDlpArgs.Cookies = &cookiesFile
	}

	isHeadlessMode, err := cmd.Flags().GetBool("headless")

	if err != nil {
		slog.Error(err.Error())
	}

	config.SetConfig(&config.Config{
		DebugDir:      &debugDir,
		CacheDisabled: isCacheDisabled,
		CacheDir:      &cacheDir,
		YtDlpArgs:     &ytDlpArgs,
		HeadlessMode:  isHeadlessMode,
		SkipOnNoMatch: configFromFile.SkipOnNoMatch,
	})

	logger := logSetup.Init(debugDir)
	defer logger.Close()

	slog.Info("starting the application")
	debsCheekResults := doAllDepsInstalled()

	var missingDeps []DebsCheckResult
	for _, dep := range debsCheekResults {
		if dep.Installed == false {
			missingDeps = append(missingDeps, dep)
		}
	}

	coreDepsPath := &youtube.CoreDepsPath{}

	if len(missingDeps) > 0 {
		for _, dep := range missingDeps {
			fmt.Printf("%s, is missing use ytmusic-tui install to install the missing dependencies ", dep.ToolName)
		}
		os.Exit(1)
	}

	for _, dep := range debsCheekResults {
		if dep.ToolName == FFmpeg {
			coreDepsPath.FFmpeg = dep.Path
		}
	}

	ins, messageChan, err := mpris.GetDbusInstance()

	if err != nil {
		slog.Error(err.Error())
	}

	backendCmd, err := startPythonBackend(debug)
	if err != nil {
		slog.Error(err.Error())
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer func() {
		if backendCmd != nil && backendCmd.Process != nil {
			_ = backendCmd.Process.Signal(syscall.SIGTERM)
			_ = backendCmd.Wait()
		}
	}()
	client := ytMusicClient.GetYtMusicClient("http://localhost:8080")
	var termWidth, termHeight int

	if !isHeadlessMode {
		termWidth, termHeight, err = term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			fmt.Println(err.Error())
			slog.Error(err.Error())
			os.Exit(1)
		}
	}

	model := ui.Model{
		BreadcrumbItems: []types.Breadcrumb{{Name: "Home", Icon: "⌂"}},
		FocusedOn:       ui.SideView,
		DBusConn:        ins,
		MainViewMode:    ui.HomePageMode,
		YtMusicClient:   client,
		CoreDepsPath:    coreDepsPath,
		BackendProcess:  backendCmd,
		Width:           termWidth - 4,
		Height:          termHeight - 4,
	}

	dims := ui.CalculateLayoutDimensions(&model)
	model.LibraryWidth = dims.SidebarWidth
	model.MainViewWidth = dims.MainWidth
	model.PlayerSectionHeight = dims.InputHeight

	model.SearchResult = list.New([]list.Item{}, ui.CustomDelegate{Model: &model}, dims.MainWidth, dims.ContentHeight)
	model.HomePageList = list.New([]list.Item{}, ui.CustomDelegate{Model: &model}, dims.MainWidth, dims.ContentHeight)
	model.RelatedList = list.New([]list.Item{}, ui.CustomDelegate{Model: &model}, dims.SidebarWidth, dims.ContentHeight)
	if isHeadlessMode {
		safeModel := ui.SafeModel{
			Model: &model,
		}
		headless.StartServer(&safeModel, messageChan)
		return nil
	}
	sideBarItems := []struct{ name, icon string }{{name: "Home", icon: "⌂"}, {name: "Library", icon: "🔖"}}
	var SideBarMenuList []list.Item
	for _, item := range sideBarItems {
		SideBarMenuList = append(SideBarMenuList, types.SidebarItem{
			Name: item.name,
			Icon: item.icon,
		})
	}
	playlistItems := list.New([]list.Item{}, ui.CustomDelegate{Model: &model}, dims.MainWidth, dims.ContentHeight)
	input := textinput.New()
	input.Placeholder = "Search tracks, artists, albums..."
	input.Prompt = "> "
	input.CharLimit = 256

	model.Alert = *bubbleup.NewAlertModel(80, true, 10*time.Second)

	model.Search = input
	musicQueueList := list.New([]list.Item{}, ui.CustomDelegate{Model: &model}, dims.SidebarWidth, dims.ContentHeight)
	model.SideBarList = list.New(SideBarMenuList, ui.CustomDelegate{Model: &model}, dims.SidebarWidth, dims.ContentHeight)

	model.SelectedPlayListItems = playlistItems
	model.MusicQueueList = &ui.MusicQueueList{
		Model:          musicQueueList,
		PaginationInfo: nil,
	}

	fgModel := ui.NewForegroundModel()
	manager := ui.Manager{
		State:        ui.Background,
		WindowWidth:  termWidth,
		WindowHeight: termHeight,
		Foreground:   fgModel,
		Background:   model,
		OverlayMode:  ui.Search,
	}
	Program := tea.NewProgram(manager, tea.WithAltScreen(), tea.WithMouseCellMotion())

	go func() {
		if messageChan == nil {
			return
		}
		for v := range *messageChan {
			Program.Send(v)
		}
	}()

	go func() {
		if types.PlayedSecondsUpdateChan == nil {
			return
		}
		for data := range types.PlayedSecondsUpdateChan {
			Program.Send(data)
		}
	}()

	_, err = Program.Run()
	if err != nil {
		slog.Error(err.Error())
		log.Fatal(err)
	}

	if ins != nil {
		ins.Conn.Close()
	}
	return nil
}

type CoreDependency string

const (
	FFmpeg  CoreDependency = "ffmpeg"
	FFprobe CoreDependency = "ffprobe"
)

type DebsCheckResult struct {
	ToolName  CoreDependency
	Installed bool
	Path      string
}

func doAllDepsInstalled() []DebsCheckResult {
	ffmpegName := "ffmpeg"
	ffprobeName := "ffprobe"
	if runtime.GOOS == "windows" {
		ffmpegName = "ffmpeg.exe"
		ffprobeName = "ffprobe.exe"
	}
	debsInCacheDirCheckPath := map[CoreDependency]string{
		FFmpeg:  filepath.Join(config.GetCacheDir(runtime.GOOS), "ffmpeg", ffmpegName),
		FFprobe: filepath.Join(config.GetCacheDir(runtime.GOOS), "ffmpeg", ffprobeName),
	}
	toolNames := []CoreDependency{FFmpeg, FFprobe}
	results := []DebsCheckResult{}
	for _, toolName := range toolNames {
		pathFound, err := exec.LookPath(string(toolName))
		if err != nil {
			isInstalledInCacheDir, err := checkDepInCacheDir(debsInCacheDirCheckPath[toolName])
			if err != nil {
				results = append(results, DebsCheckResult{
					ToolName:  toolName,
					Installed: false,
					Path:      "",
				})
				continue
			}
			if isInstalledInCacheDir {
				results = append(results, DebsCheckResult{
					ToolName:  toolName,
					Installed: true,
					Path:      debsInCacheDirCheckPath[toolName],
				})
			}
			continue
		}
		results = append(results, DebsCheckResult{
			ToolName:  toolName,
			Installed: true,
			Path:      pathFound,
		})
	}
	return results
}

func checkDepInCacheDir(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		slog.Error(err.Error())
		return false, err
	}

	if fileInfo.IsDir() {
		err := errors.New("the provided path is not a valid file")
		slog.Error(err.Error())
		return false, err
	}

	return true, nil
}

var (
	cpuFile string
	memFile string
)

func Execute(version string, debug bool) error {
	cmd := newRootCmd(version, debug)
	if debug {
		cmd.PersistentFlags().StringVar(&cpuFile, "cpuprofile", "", "write cpu profile to `file`")
		cmd.PersistentFlags().StringVar(&memFile, "memprofile", "", "write memory profile to `file`")

		cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			if cpuFile != "" {
				f, err := os.Create(cpuFile)
				if err != nil {
					return fmt.Errorf("could not create CPU profile: %w", err)
				}
				if err := pprof.StartCPUProfile(f); err != nil {
					f.Close()
					return fmt.Errorf("could not start CPU profile: %w", err)
				}
			}
			return nil
		}
	}

	defaultDebugDir := filepath.Join(config.GetStateDir(runtime.GOOS), "logs")
	cmd.Flags().StringP("debug-dir", "d", defaultDebugDir, "a path to store app logs")
	cmd.Flags().StringP("cache-dir", "c", config.GetCacheDir(runtime.GOOS), "a path to store app cache")
	cmd.Flags().Bool("disable-cache", false, "disable cache")
	cmd.Flags().Bool("headless", false, "Headless mode which provides api endpoint to build custom ui")
	cmd.Flags().String("cookies-from-browser", "", "The name of the browser to load cookies from this option is used by yt-dlp see yt-dlp docs to see supported browsers")
	cmd.Flags().String("cookies", "", "cookies file the option you pass for this flag will be passed to yt-dlp checkout yt-dlp docs to learn more about this flag")

	if err := cmd.Execute(); err != nil {
		return fmt.Errorf("error executing root command: %w", err)
	}
	return nil
}
