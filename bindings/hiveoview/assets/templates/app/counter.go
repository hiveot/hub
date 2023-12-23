package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/assets/templates/layouts"
	"html/template"
	"net/http"
)

var counter int = 0

func InitCounterComp(t *template.Template, r chi.Router) {
	r.Get("/htmx/counter.html", func(w http.ResponseWriter, r *http.Request) {
		counter++
		data := map[string]any{
			"value": counter,
		}
		layouts.Render(w, t, "counter.html", data)
	})

}
