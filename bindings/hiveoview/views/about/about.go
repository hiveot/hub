package about

import (
	"github.com/hiveot/hub/bindings/hiveoview/views/layouts"
	"html/template"
	"net/http"
)

// GetAbout renders the about view
func GetAbout(t *template.Template) http.HandlerFunc {
	//mt := t.Lookup("main.html")
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{}
		layouts.RenderWithLayout(w, t, "about.html", "main.html", data)
	}
}
