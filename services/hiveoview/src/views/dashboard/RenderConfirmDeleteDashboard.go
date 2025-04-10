package dashboard

import (
	"fmt"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"log/slog"
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
		Dashboard:                 cdc.CurrentDashboard(),
		SubmitDeleteDashboardPath: getDashboardPath(src.DeleteDashboardPath, cdc),
	}
	buff, err := app.RenderAppOrFragment(r, RenderConfirmDeleteDashboardTemplate, data)
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

	// navigate back to the default dashboard. Notes:
	// 1. http.Redirect doesn't work
	// 2. Setting the HX-Redirect header does but does a full page reload.
	// 3. hx-location works with boost, but needs a target:
	//    See https://htmx.org/headers/hx-location/
	// todo; standardize this type of navigation along with the templates
	w.Header().Add("HX-Location", fmt.Sprintf(
		"{\"path\":\"%s\", \"target\":\"%s\"}",
		src.RenderDashboardRootPath, "#dashboardPage"))

	w.WriteHeader(http.StatusOK)
}
