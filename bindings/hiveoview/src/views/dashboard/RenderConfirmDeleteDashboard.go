package dashboard

import (
	"github.com/hiveot/hub/bindings/hiveoview/src"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const RenderConfirmDeleteDashboardTemplate = "RenderConfirmDeleteDashboard.gohtml"

type ConfirmDeleteDashboardTemplateData struct {
	Dashboard                 session.DashboardModel
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
		SubmitDeleteDashboardPath: getDashboardPath(src.DeleteDashboardPath, cdc),
	}
	buff, err := app.RenderAppOrFragment(r, RenderConfirmDeleteDashboardTemplate, data)
	sess.WritePage(w, buff, err)
}
