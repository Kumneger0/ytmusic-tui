//go:build linux && amd64

package backend

import "embed"

//go:embed binaries/python-linux-amd64
var PythonBackend embed.FS

// PythonBacked is an alias for PythonBackend for backwards compatibility.
var PythonBacked = PythonBackend

const embedFilePath = "binaries/python-linux-amd64"
