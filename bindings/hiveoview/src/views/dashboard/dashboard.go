package dashboard

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/views"
	"net/http"
)

// RenderDashboard renders the dashboard fragment
func RenderDashboard(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	views.TM.RenderTemplate(w, r, "dashboard.html", data)
}
