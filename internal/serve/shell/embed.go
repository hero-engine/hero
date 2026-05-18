// Package shell hosts the slim web-app chrome (top nav, page router,
// shared fragments) that every hero serve home rides on. The shell
// owns its own embedded assets — templates and static CSS / SVG — so
// no other package needs to know how they're packaged.
package shell

import (
	"embed"
	"io/fs"
)

//go:embed templates/*.html static/* static/islands/*
var assetsFS embed.FS

// StaticFS returns the read-only filesystem rooted at the shell's
// `static/` directory. The serve package mounts this at
// `/static/shell/` so page-layout.html can reference assets at stable
// paths.
func StaticFS() fs.FS {
	sub, err := fs.Sub(assetsFS, "static")
	if err != nil {
		// embed.FS always has the declared directory; panic = bug.
		panic("shell: static subdir missing from embed: " + err.Error())
	}
	return sub
}

// templatesFS exposes the embedded templates directory for the
// internal template loader.
func templatesFS() fs.FS {
	sub, err := fs.Sub(assetsFS, "templates")
	if err != nil {
		panic("shell: templates subdir missing from embed: " + err.Error())
	}
	return sub
}
