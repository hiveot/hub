package assets

import (
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
)

// EmbeddedViews and EmbeddedComponents contains all html templates
//
//go:embed views/*
var EmbeddedViews embed.FS

//go:embed components/*
var EmbeddedComponents embed.FS

// EmbeddedStatic contains all static assets
//
//go:embed static/* components/*
var EmbeddedStatic embed.FS

var AllTemplates, _ = parseTemplates()

// ParseTemplates the html templates of the embedded filesystem
func parseTemplates() (*template.Template, error) {
	t := template.New("hiveot")
	// embed app templates first to allow components to override templates in block statements
	err := parseTemplateFiles(t, EmbeddedViews)
	if err == nil {
		err = parseTemplateFiles(t, EmbeddedComponents)
	}
	return t, err
}

// ParseTemplateFiles the html templates of the given filesystem
func parseTemplateFiles(t *template.Template, files fs.FS) error {
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

// GetTemplate returns a Clone of the template by name, ready for executing
// Returns nil of template doesn't exist.
func GetTemplate(name string) *template.Template {
	tpl := AllTemplates.Lookup(name)
	if tpl == nil {
		slog.Error("Template not found", "name", name)
		return nil
	}
	tpl, err := tpl.Clone()
	if err != nil {
		slog.Error("Clone template failed", "err", err)
	}
	return tpl
}
