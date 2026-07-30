//go:build darwin && arm64

package backend

import "embed"

//go:embed binaries/python-darwin-arm64
var PythonBackend embed.FS

const embedFilePath = "binaries/python-darwin-arm64"
