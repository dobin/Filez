// Package web holds embedded templates and static assets so the binary is self-contained.
package web

import (
	"embed"
	"io/fs"
)

//go:embed templates/*.html
var TemplatesFS embed.FS

//go:embed static/*
var StaticFS embed.FS

// StaticSubFS returns the embedded static subtree for http.FileServer.
func StaticSubFS() fs.FS {
	sub, _ := fs.Sub(StaticFS, "static")
	return sub
}
