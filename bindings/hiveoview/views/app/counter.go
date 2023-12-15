package app

import (
	"github.com/go-chi/chi/v5"
	"html/template"
	"log/slog"
	"net/http"
)

var counter int = 0

func InitCounterComp(t *template.Template, r chi.Router) {
	r.Get("/htmx/counter.html", func(w http.ResponseWriter, r *http.Request) {
		counter++
		data := map[string]any{
			"value": counter,
		}
		err := t.ExecuteTemplate(w, "counter", data)
		if err != nil {
			slog.Error("Error rendering template", "err", err)
		}
	})

}
