#!/bin/sh
set -e
SERVER_URL="${1:-$SERVER_URL}"
rm -rf completions
mkdir completions
for sh in bash zsh fish; do
	go run -ldflags "-X main.serverURL=${SERVER_URL}" main.go completion "$sh" >"completions/ytmusic-tui.$sh"
done