package assets

import (
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"sync"
)

var renderMux sync.Mutex

// RenderTemplate looks-up the template 'name' and renders it with the given data.
//
//	name is the name of the template to render
//	data is a map with template variables and their values
func RenderTemplate(w http.ResponseWriter, name string, data map[string]any) {
	renderMux.Lock()
	defer renderMux.Unlock()
	tpl := GetTemplate(name)
	if tpl == nil || tpl.Tree == nil {
		err := errors.New("missing or invalid template: " + name)
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error(err.Error())
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	tpl, err := tpl.Clone()
	if err == nil {
		err = tpl.Execute(w, data)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("rendering template failed", "err", err)
		_, _ = w.Write([]byte("template render error: " + err.Error()))
		return
	}
}

// RenderWithLayout embeds the template 'name' into the overlay and executes.
// The overlay must have an 'embed' field.
// If a template has an error, the error is returned to the user instead along with a 500 error.
//
//	name is the name of the template to render
//	overlay is the optional overlay to use. "" for the default overlay layout.html.
func RenderWithLayout(w http.ResponseWriter, t *template.Template, name string, overlay string, data map[string]any) {
	renderMux.Lock()
	defer renderMux.Unlock()

	if overlay == "" {
		overlay = "layout.html"
	}
	overlayT := t.Lookup(overlay)
	overlayT, err := overlayT.Clone()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("Cloning overlay failed", "err", err)
		_, _ = w.Write([]byte("overlay error: " + err.Error()))
		return
	}
	tpl := t.Lookup(name)
	if tpl == nil || tpl.Tree == nil {
		err = errors.New("missing or invalid template: " + name)
	} else {
		// problem with error "cannot Clone ... after it has executed"
		tpl, err = tpl.Clone()
		// This is where the magic happens: replace the 'embed' template with the given template.
		if err == nil {
			_, err = overlayT.AddParseTree("embed", tpl.Tree)
		}
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("merging templates failed", "err", err)
		_, _ = w.Write([]byte("template error: " + err.Error()))
		return
	}
	err = overlayT.Execute(w, data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("rendering template failed", "err", err)
		_, _ = w.Write([]byte("template render error: " + err.Error()))
		return
	}
}
