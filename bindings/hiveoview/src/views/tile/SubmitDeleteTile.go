package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src"
	"log/slog"
	"net/http"
	"strings"
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

	// Notify the UI that the tile has been removed.
	eventName := strings.ReplaceAll(src.DashboardUpdatedEvent, "{dashboardID}", ctc.dashboardID)
	sess.SendSSE(eventName, "")

	sess.WriteError(w, err, http.StatusOK)
}
