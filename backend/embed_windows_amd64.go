//go:build windows && amd64

package backend

import "embed"

//go:embed binaries/python-windows-amd64.exe
var PythonBackend embed.FS

const embedFilePath = "binaries/python-windows-amd64.exe"
