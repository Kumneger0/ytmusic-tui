//go:build darwin && arm64

package backend

import "embed"

//go:embed binaries/python-darwin-arm64
var PythonBackend embed.FS

// PythonBacked is an alias for PythonBackend for backwards compatibility.
var PythonBacked = PythonBackend

const embedFilePath = "binaries/python-darwin-arm64"
