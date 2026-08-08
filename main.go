package main

import (
	"log/slog"

	"github.com/kumneger0/ytmusic-tui/cmd"
)

var (
	version   = ""
	Debug     = "false"
	serverURL = ""
)

func main() {
	if serverURL == "" {
		panic("server url is missing")
	}
	err := cmd.Execute(version, Debug == "true", serverURL)
	if err != nil {
		slog.Error(err.Error())
	}
}
