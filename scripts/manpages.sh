#!/bin/sh
set -e
SERVER_URL="${1:-$SERVER_URL}"
rm -rf manpages
mkdir manpages
go run -ldflags "-X main.serverURL=${SERVER_URL}" . man | gzip -c -9 >manpages/ytmusic-tui.1.gz