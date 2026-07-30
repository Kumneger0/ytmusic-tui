//go:build windows && amd64

package backend

import "embed"

//go:embed binaries/python-windows-amd64.exe
var PythonBackend embed.FS

// PythonBacked is an alias for PythonBackend for backwards compatibility.
var PythonBacked = PythonBackend

const embedFilePath = "binaries/python-windows-amd64.exe"
