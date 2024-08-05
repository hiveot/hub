package dashboard

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const ConfigDashboardTemplateFile = "RenderConfigDashboard.gohtml"
const SubmitConfigDashboardPath = "/dashboard/{dashboardID}/config" // POST

type ConfigDashboardTemplateData struct {
	Dashboard             session.DashboardDefinition
	SubmitConfigDashboard string
}

// RenderConfigDashboard renders the dialog for configuring a dashboard
func RenderConfigDashboard(w http.ResponseWriter, r *http.Request) {
	cs, cdc, err := getDashboardContext(r, true)
	if err != nil {
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}
	data := ConfigDashboardTemplateData{
		Dashboard:             cdc.dashboard,
		SubmitConfigDashboard: getDashboardPath(SubmitConfigDashboardPath, cdc),
	}
	app.RenderAppOrFragment(w, r, ConfigDashboardTemplateFile, data)
}
