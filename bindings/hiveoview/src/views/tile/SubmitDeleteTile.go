package tile

import (
	"log/slog"
	"net/http"
)

// SubmitDeleteTile removes a tile from the dashboard
// Right now it is assumed that a tile is only used in a single dashboard
func SubmitDeleteTile(w http.ResponseWriter, r *http.Request) {
	sess, ctc, err := GetTileContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	err = r.ParseForm()
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	// delete the dashboard
	slog.Info("SubmitDashboardDeleteTile",
		slog.String("clientID", ctc.clientID),
		slog.String("dashboardID", ctc.dashboardID),
		slog.String("tileID", ctc.tileID),
	)
	delete(ctc.dashboard.Tiles, ctc.tileID)
	ctc.clientModel.UpdateDashboard(&ctc.dashboard)
	err = sess.SaveState()

	sess.WriteError(w, err, http.StatusOK)
}
