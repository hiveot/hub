package dashboard

import (
	"fmt"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"log/slog"
	"net/http"
)

const HandleRenderDefaultDashboardPath = "/dashboard"

// SubmitDeleteDashboard applies deleting a dashboard
func SubmitDeleteDashboard(w http.ResponseWriter, r *http.Request) {
	cs, cdc, err := getDashboardContext(r, false)
	if err != nil {
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}

	// delete the dashboard
	slog.Info("Deleting Dashboard", slog.String("clientID", cdc.clientID))
	cdc.clientModel.DeleteDashboard(cdc.dashboardID)

	msgText := fmt.Sprintf("Dashboard '%s' removed from the directory", cdc.dashboardID)
	cs.SendNotify(session.NotifySuccess, msgText)
	// navigate back to the default dashboard.
	// http.Redirect doesn't work but using HX-Redirect header does.
	// see also: https://www.reddit.com/r/htmx/comments/188oqx5/htmx_form_submission_issue_redirecting_on_success/
	//http.Redirect(w, r, "/app/dashboard", http.StatusMovedPermanently)
	w.Header().Add("HX-Redirect", HandleRenderDefaultDashboardPath)
	w.WriteHeader(http.StatusOK)
}
