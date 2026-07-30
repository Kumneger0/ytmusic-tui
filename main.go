package main

import (
	"log/slog"

	"github.com/kumneger0/ytmusic-tui/cmd"
)

var (
	version = ""
	Debug   = "false"
)

func main() {
	err := cmd.Execute(version, Debug == "true")
	if err != nil {
		slog.Error(err.Error())
	}
}
