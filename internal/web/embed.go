package web

import (
	"embed"
	"io/fs"
	"path"
)

var (
	//go:embed static/*
	embeddedStaticFiles embed.FS

	//go:embed templates/*
	embeddedTemplates embed.FS
)

// templateEmbedFS is a wrapper around embed.FS to implement fs.FS interface
// for the 'templates' directory.
type templateEmbedFS struct {
	content embed.FS
}

// Open opens the named file from the 'templates' directory.
func (e templateEmbedFS) Open(name string) (fs.File, error) {
	return e.content.Open(path.Join("templates", name))
}
