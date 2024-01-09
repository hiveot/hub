package thingsview

import (
	"github.com/hiveot/hub/bindings/hiveoview/assets"
	"net/http"
)

// RenderThings renders the things view.
//
// This requires a connection to the Hub.
func RenderThings(w http.ResponseWriter, r *http.Request) {

	data := map[string]any{}
	assets.RenderTemplate(w, "things.html", data)
}
