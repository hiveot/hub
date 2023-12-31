package about

import (
	"github.com/hiveot/hub/bindings/hiveoview/assets"
	"net/http"
)

// RenderAbout renders the about view
func RenderAbout(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"By":      "The Hive",
		"Version": "pre-alpha",
	}
	// simply render this as full or partial
	assets.RenderMain(w, r, "about.html", data)
}
