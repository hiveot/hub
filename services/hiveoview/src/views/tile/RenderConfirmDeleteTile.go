package tile

import (
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"log/slog"
	"net/http"
	"strings"
)

const ConfirmDeleteTileTemplate = "RenderConfirmDeleteTile.gohtml"

type ConfirmDeleteTileTemplateData struct {
	Dashboard            session.DashboardModel
	Tile                 session.DashboardTile
	SubmitDeleteTilePath string
}

func RenderConfirmDeleteTile(w http.ResponseWriter, r *http.Request) {

	sess, ctc, err := GetTileContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	data := ConfirmDeleteTileTemplateData{
		Dashboard:            ctc.dashboard,
		Tile:                 ctc.tile,
		SubmitDeleteTilePath: getTilePath(src.PostTileDeletePath, ctc),
	}
	buff, err := app.RenderAppOrFragment(r, ConfirmDeleteTileTemplate, data)
	sess.WritePage(w, buff, err)
}

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

	// Notify the UI that the tile has been removed.
	eventName := strings.ReplaceAll(src.DashboardUpdatedEvent, "{dashboardID}", ctc.dashboardID)
	sess.SendSSE(eventName, "")

	sess.WriteError(w, err, http.StatusOK)
}
