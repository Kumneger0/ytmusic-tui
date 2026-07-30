//go:build linux

package backend

import "embed"

//go:embed python-linux
var PythonBackend embed.FS

// PythonBacked is an alias for PythonBackend for backwards compatibility.
var PythonBacked = PythonBackend

const embedFileName = "python-linux"
