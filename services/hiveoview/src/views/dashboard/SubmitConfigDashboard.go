package dashboard

import (
	"log/slog"
	"net/http"
)

// SubmitConfigDashboard updates the dashboard configuration
func SubmitConfigDashboard(w http.ResponseWriter, r *http.Request) {

	sess, cdc, err := getDashboardContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	newTitle := r.PostFormValue("title")

	slog.Info("SubmitConfigDashboard", "SenderID", cdc.clientID,
		"dashboardID", cdc.dashboardID)

	cdc.dashboard.Title = newTitle
	cdc.clientModel.UpdateDashboard(&cdc.dashboard)
	_ = sess.SaveState()

	// refresh the dashboard
	w.WriteHeader(http.StatusOK)
}
