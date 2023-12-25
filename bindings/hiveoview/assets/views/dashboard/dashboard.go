package dashboard

import (
	"github.com/hiveot/hub/bindings/hiveoview/assets"
	"net/http"
)

// RenderDashboard renders the dashboard fragment
func RenderDashboard(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	assets.RenderTemplate(w, "dashboard.html", data)
}
