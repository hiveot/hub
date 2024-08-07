package dashboard

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const RenderConfirmDeleteDashboardTemplate = "RenderConfirmDeleteDashboard.gohtml"
const SubmitDeleteDashboardPath = "/dashboard/{dashboardID}"

type ConfirmDeleteDashboardTemplateData struct {
	Dashboard                 session.DashboardDefinition
	SubmitDeleteDashboardPath string
}

// RenderConfirmDeleteDashboard renders confirmation dialog for deleting a dashboard
func RenderConfirmDeleteDashboard(w http.ResponseWriter, r *http.Request) {

	sess, cdc, err := getDashboardContext(r, false)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// setup the rendering data
	data := ConfirmDeleteDashboardTemplateData{
		Dashboard:                 cdc.dashboard,
		SubmitDeleteDashboardPath: getDashboardPath(SubmitDeleteDashboardPath, cdc),
	}
	buff, err := app.RenderAppOrFragment(r, RenderConfirmDeleteDashboardTemplate, data)
	sess.WritePage(w, buff, err)
}
