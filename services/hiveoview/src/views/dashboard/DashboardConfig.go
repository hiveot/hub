package dashboard

import (
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"log/slog"
	"net/http"
)

const RenderDashboardConfigTemplateFile = "DashboardConfig.gohtml"

type RenderDashboardConfigTemplateData struct {
	Dashboard             session.DashboardModel
	SubmitConfigDashboard string
}

// RenderDashboardConfig renders the dialog for configuring a dashboard
func RenderDashboardConfig(w http.ResponseWriter, r *http.Request) {
	sess, cdc, err := getDashboardContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	data := RenderDashboardConfigTemplateData{
		Dashboard:             cdc.CurrentDashboard(),
		SubmitConfigDashboard: getDashboardPath(src.PostDashboardConfigPath, cdc),
	}
	buff, err := app.RenderAppOrFragment(r, RenderDashboardConfigTemplateFile, data)
	sess.WritePage(w, buff, err)
}

// SubmitDashboardConfig updates the dashboard configuration
func SubmitDashboardConfig(w http.ResponseWriter, r *http.Request) {

	sess, cdc, err := getDashboardContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// 'title' is the form field from gohtml
	newTitle := r.PostFormValue("title")

	slog.Info("SubmitDashboardConfig", "SenderID", cdc.clientID,
		"dashboardID", cdc.dashboardID)

	dashboard := cdc.CurrentDashboard()
	dashboard.Title = newTitle
	cdc.clientModel.UpdateDashboard(&dashboard)

	// refresh the dashboard
	w.WriteHeader(http.StatusOK)
}
