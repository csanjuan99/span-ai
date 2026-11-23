package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var files embed.FS

// Pages is the built desk and span page.
func Pages() (fs.FS, error) {
	return fs.Sub(files, "dist")
}
