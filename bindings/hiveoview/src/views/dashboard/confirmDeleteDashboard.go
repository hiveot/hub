package dashboard

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"log/slog"
	"net/http"
)

type ConfirmDeleteDashboardTemplateData struct {
	Dashboard session.DashboardDefinition
}

func RenderConfirmDeleteDashboard(w http.ResponseWriter, r *http.Request) {
	dashboardID := chi.URLParam(r, "page")

	cs, _, err := session.GetSessionFromContext(r)
	if err != nil {
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}
	clientModel := cs.GetClientData()
	dashboard, found := clientModel.GetDashboard(dashboardID)
	if !found {
		cs.WriteError(w, fmt.Errorf("Dashboard '%s' not found", dashboardID), http.StatusBadRequest)
		return
	}
	data := ConfirmDeleteDashboardTemplateData{
		Dashboard: dashboard,
	}
	app.RenderAppOrFragment(w, r, "confirmDeleteDashboardDialog.gohtml", data)
}

// HandleDeleteDashboard handles removal of a thing TD document
func HandleDeleteDashboard(w http.ResponseWriter, r *http.Request) {
	dashboardID := chi.URLParam(r, "page")

	// get the hub client connection and read the existing TD
	mySession, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		mySession.WriteError(w, err, http.StatusBadRequest)
		return
	}

	// delete the dashboard
	slog.Info("Deleting Dashboard", slog.String("clientID", hc.ClientID()))
	clientModel := mySession.GetClientData()
	clientModel.DeleteDashboard(dashboardID)

	msgText := fmt.Sprintf("Dashboard '%s' successfully removed from the directory", dashboardID)
	mySession.SendNotify(session.NotifySuccess, msgText)
	// navigate back to the default dashboard.
	// http.Redirect doesn't work but using HX-Redirect header does.
	// see also: https://www.reddit.com/r/htmx/comments/188oqx5/htmx_form_submission_issue_redirecting_on_success/
	//http.Redirect(w, r, "/app/dashboard", http.StatusMovedPermanently)
	w.Header().Add("HX-Redirect", "/app/dashboard")
	w.WriteHeader(http.StatusOK)
}
