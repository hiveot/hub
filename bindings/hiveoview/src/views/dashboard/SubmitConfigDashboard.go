package dashboard

import (
	"log/slog"
	"net/http"
)

// SubmitConfigDashboard updates a the dashboard configuration
func SubmitConfigDashboard(w http.ResponseWriter, r *http.Request) {

	cs, cdc, err := getDashboardContext(r, true)
	if err != nil {
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}

	newTitle := r.PostFormValue("title")

	slog.Info("SubmitConfigDashboard", "ClientID", cdc.clientID,
		"dashboardID", cdc.dashboardID)

	cdc.dashboard.Title = newTitle
	cdc.clientModel.UpdateDashboard(&cdc.dashboard)
	err = cs.SaveState()

	// refresh the dashboard
	w.WriteHeader(http.StatusOK)
}
