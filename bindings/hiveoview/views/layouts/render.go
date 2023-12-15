package layouts

import (
	"html/template"
	"log/slog"
	"net/http"
)

// RenderWithLayout embeds the template 'name' into the overlay and executes
// The overlay must have an 'embed' field.
func RenderWithLayout(w http.ResponseWriter, t *template.Template, name string, overlay string, data map[string]any) {
	overlayT, err := t.Lookup(overlay).Clone()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("Cloning overlay failed", "err", err)
		return
	}
	tpl := t.Lookup(name)
	_, err = overlayT.AddParseTree("embed", tpl.Tree)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("merging templates failed", "err", err)
		return
	}
	err = overlayT.Execute(w, data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("rendering template failed", "err", err)
		return
	}
}
