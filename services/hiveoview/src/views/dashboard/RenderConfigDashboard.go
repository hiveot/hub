package dashboard

import (
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"net/http"
)

const ConfigDashboardTemplateFile = "RenderConfigDashboard.gohtml"

type ConfigDashboardTemplateData struct {
	Dashboard             session.DashboardModel
	SubmitConfigDashboard string
}

// RenderConfigDashboard renders the dialog for configuring a dashboard
func RenderConfigDashboard(w http.ResponseWriter, r *http.Request) {
	sess, cdc, err := getDashboardContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	data := ConfigDashboardTemplateData{
		Dashboard:             cdc.dashboard,
		SubmitConfigDashboard: getDashboardPath(src.PostDashboardConfigPath, cdc),
	}
	buff, err := app.RenderAppOrFragment(r, ConfigDashboardTemplateFile, data)
	sess.WritePage(w, buff, err)
}
