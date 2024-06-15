package status

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const TemplateFile = "status.gohtml"

// RenderStatus renders the client status page
func RenderStatus(w http.ResponseWriter, r *http.Request) {
	status := app.GetConnectStatus(r)

	data := map[string]any{}
	data["Progress"] = status

	// full render or fragment render
	app.RenderAppOrFragment(w, r, TemplateFile, data)
}
