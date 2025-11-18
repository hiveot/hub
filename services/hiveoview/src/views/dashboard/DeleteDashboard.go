package dashboard

import (
	"fmt"
	"github.com/hiveot/hivehub/services/hiveoview/src"
	"github.com/hiveot/hivehub/services/hiveoview/src/session"
	"github.com/hiveot/hivehub/services/hiveoview/src/views/app"
	"log/slog"
	"net/http"
)

const RenderDeleteDashboardTemplate = "DeleteDashboard.gohtml"

type DeleteDashboardTemplateData struct {
	Dashboard                 session.DashboardModel
	SubmitDeleteDashboardPath string
}

// RenderDeleteDashboard renders confirmation dialog for deleting a dashboard
func RenderDeleteDashboard(w http.ResponseWriter, r *http.Request) {

	sess, cdc, err := getDashboardContext(r, false)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	// setup the rendering data
	dashboard, _ := cdc.SelectedDashboard()
	data := DeleteDashboardTemplateData{
		Dashboard:                 dashboard,
		SubmitDeleteDashboardPath: getDashboardPath(src.PostDashboardDeletePath, cdc),
	}
	buff, err := app.RenderAppOrFragment(r, RenderDeleteDashboardTemplate, data)
	sess.WritePage(w, buff, err)
}

// SubmitDeleteDashboard applies deleting a dashboard
func SubmitDeleteDashboard(w http.ResponseWriter, r *http.Request) {
	sess, cdc, err := getDashboardContext(r, false)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	// delete the dashboard
	slog.Info("Deleting Dashboard", slog.String("clientID", cdc.clientID))
	cdc.clientModel.DeleteDashboard(cdc.dashboardID)

	msgText := fmt.Sprintf("Dashboard '%s' removed from the directory", cdc.dashboardID)
	sess.SendNotify(session.NotifySuccess, "", msgText)

	// need a full page reload for the menu to update list of dashboards
	w.Header().Add("HX-Redirect", src.RenderDashboardRootPath)
	w.WriteHeader(http.StatusOK)
}
