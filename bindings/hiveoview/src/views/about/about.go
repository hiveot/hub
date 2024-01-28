package about

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

func RenderAbout(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	app.RenderAppOrFragment(w, r, "about.html", data)
}
