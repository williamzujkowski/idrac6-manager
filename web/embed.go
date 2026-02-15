// Package web embeds the static web assets.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:static
var content embed.FS

// FS returns the embedded filesystem rooted at the static directory.
func FS() fs.FS {
	sub, err := fs.Sub(content, "static")
	if err != nil {
		panic("failed to access embedded web content: " + err.Error())
	}
	return sub
}
