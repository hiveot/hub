package tile

import (
	"encoding/json"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"log/slog"
	"net/http"
)

// SubmitEditTile updated or creates a tile and adds it to the dashboard
func SubmitEditTile(w http.ResponseWriter, r *http.Request) {

	sess, cdc, err := GetTileContext(r, false)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	err = r.ParseForm()

	slog.Info("SubmitEditTile",
		slog.String("clientID", cdc.clientID),
		slog.String("dashboardID", cdc.dashboardID),
		slog.String("tileID", cdc.tileID),
	)
	newTitle := r.FormValue("title")
	tileType := r.FormValue("tileType")
	sources := r.FormValue("sources")
	_ = sources

	tile := cdc.dashboard.NewTile(cdc.tileID, "", session.TileTypeText)
	tile.Title = newTitle
	tile.TileType = tileType
	// TODO: get list of sources from the form
	if sources != "" {
		err = json.Unmarshal([]byte(sources), &tile.Sources)
	}

	// add the new tile to the dashboard
	cdc.dashboard.Tiles[tile.ID] = tile
	cdc.clientModel.UpdateDashboard(&cdc.dashboard)
	// save the new dashboard and tile
	err = sess.SaveState()

	// TODO: sse notify change of tile so it is re-rendered

	sess.WriteError(w, err, http.StatusOK)
}
