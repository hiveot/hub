package assets

import (
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
)

// EmbeddedTemplates contains all assets
//
//go:embed templates/*
var EmbeddedTemplates embed.FS

//go:embed components/*
var EmbeddedComponents embed.FS

// EmbeddedStatic contains all static assets
//
//go:embed static/*
var EmbeddedStatic embed.FS

// ParseTemplates the html templates of the embedded filesystem
func ParseTemplates() (*template.Template, error) {
	t := template.New("hiveot")
	// embed app templates first to allow components to override templates in block statements
	err := ParseTemplateFiles(t, EmbeddedTemplates)
	if err == nil {
		err = ParseTemplateFiles(t, EmbeddedComponents)
	}

	return t, err
}

// ParseTemplateFiles the html templates of the given filesystem
func ParseTemplateFiles(t *template.Template, files fs.FS) error {
	err := fs.WalkDir(files, ".", func(parent string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			subT, err := t.ParseFS(files, filepath.Join(parent, "*.html"))
			_ = subT
			// only report template errors, but keep going
			if err != nil && !strings.HasPrefix(err.Error(), "template: pattern matches no files:") {
				slog.Error("error parsing template", "err", err)
			}
			err = nil
		}
		return err
	})
	return err
}
