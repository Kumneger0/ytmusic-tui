//go:build darwin

package backend

import "embed"

//go:embed python-darwin
var PythonBackend embed.FS

// PythonBacked is an alias for PythonBackend for backwards compatibility.
var PythonBacked = PythonBackend

const embedFileName = "python-darwin"
