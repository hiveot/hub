package about

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const TemplateFile = "about.gohtml"

func RenderAbout(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	app.RenderAppOrFragment(w, r, TemplateFile, data)
}
