package thingsview

import (
	"html/template"
	"log/slog"
	"net/http"
)

// GetThings renders the things view.
//
// This requires a connection to the Hub.
func GetThings(t *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		data := map[string]any{}
		err := t.ExecuteTemplate(w, "things.html", data)
		if err != nil {
			slog.Error("Error rendering template", "err", err)
		}
	}
}
