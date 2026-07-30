//go:build windows

package backend

import "embed"

//go:embed python-windows.exe
var PythonBackend embed.FS

// PythonBacked is an alias for PythonBackend for backwards compatibility.
var PythonBacked = PythonBackend

const embedFileName = "python-windows.exe"
