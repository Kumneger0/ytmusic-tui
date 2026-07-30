$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Variables
$repo = "kumneger0/ytmusic-tui"
$ytmusic-tuiDir = "$env:LOCALAPPDATA\ytmusic-tui"
$binDir = "$ytmusic-tuiDir\bin"
$exe = "$binDir\ytmusic-tui.exe"

# Functions
function Write-Success {
    param($Message)
    Write-Host " > $Message" -ForegroundColor 'Green'
}

function Write-Info {
    param($Message)
    Write-Host " > $Message" -ForegroundColor 'Cyan'
}

function Test-Admin {
    $currentUser = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    $currentUser.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Checks
if (Test-Admin) {
    Write-Warning "The script is running as administrator. It is recommended to install ytmusic-tui as a regular user."
    $choices = [System.Management.Automation.Host.ChoiceDescription[]] @(
        (New-Object System.Management.Automation.Host.ChoiceDescription '&Yes', 'Abort installation.'),
        (New-Object System.Management.Automation.Host.ChoiceDescription '&No', 'Resume installation.')
    )
    $choice = $Host.UI.PromptForChoice('Warning', 'Do you want to abort the installation process?', $choices, 0)
    if ($choice -eq 0) {
        Write-Host 'Installation aborted.' -ForegroundColor 'Yellow'
        exit
    }
}

# Determine Architecture
if ($env:PROCESSOR_ARCHITECTURE -eq 'AMD64') {
    $target = "Windows_x86_64"
}
elseif ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {
    $target = "Windows_arm64"
}
elseif ($env:PROCESSOR_ARCHITECTURE -eq 'x86') {
    $target = "Windows_386"
}
else {
    Write-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
    exit
}

# Fetch Version
if ($v) {
    $version = $v.Replace('v', '')
}
else {
    try {
        $latestRelease = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
        $version = $latestRelease.tag_name.Replace('v', '')
    }
    catch {
        Write-Error "Failed to fetch latest version from GitHub API: $($_.Exception.Message)"
        Write-Host "Try passing the version manually, for example:" -ForegroundColor 'Yellow'
        Write-Host '$v = "0.4.0"; Get-Content .\install.ps1 -Raw | iex' -ForegroundColor 'Yellow'
        exit
    }
}

Write-Info "Installing ytmusic-tui v$version for $target..."

# Download
$archivePath = [System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), "ytmusic-tui.tar.gz")
$downloadUrl = "https://github.com/$repo/releases/download/v$version/ytmusic-tui_$($target).tar.gz"

Write-Info "Downloading from $downloadUrl..."
Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing

# Install
if (-not (Test-Path $binDir)) {
    New-Item -Path $binDir -ItemType Directory -Force | Out-Null
}

Write-Info "Extracting to $binDir..."
tar -xzf $archivePath -C $binDir

# Cleanup
Remove-Item -Path $archivePath -Force -ErrorAction SilentlyContinue

# PATH Update
Write-Info "Adding ytmusic-tui to PATH..."
$userPath = [Environment]::GetEnvironmentVariable('PATH', [EnvironmentVariableTarget]::User)
if ($userPath -notlike "*$binDir*") {
    $newPath = "$userPath;$binDir"
    [Environment]::SetEnvironmentVariable('PATH', $newPath, [EnvironmentVariableTarget]::User)
    $env:PATH = "$env:PATH;$binDir"
    Write-Success "ytmusic-tui added to User PATH."
}
else {
    Write-Info "ytmusic-tui is already in PATH."
}

Write-Success "ytmusic-tui v$version was successfully installed!"
Write-Host "Restart your terminal to start using 'ytmusic-tui'."
