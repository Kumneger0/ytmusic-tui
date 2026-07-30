//go:build linux && amd64

package backend

import "embed"

//go:embed binaries/python-linux-amd64
var PythonBackend embed.FS

const embedFilePath = "binaries/python-linux-amd64"
