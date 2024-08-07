package tile

import (
	"encoding/json"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"log/slog"
	"net/http"
)

// SubmitConfigTile updated or creates a tile and adds it to the dashboard
func SubmitConfigTile(w http.ResponseWriter, r *http.Request) {

	sess, cdc, err := GetTileContext(r, false)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	err = r.ParseForm()

	slog.Info("SubmitConfigTile",
		slog.String("clientID", cdc.clientID),
		slog.String("dashboardID", cdc.dashboardID),
		slog.String("tileID", cdc.tileID),
	)

	tile := cdc.clientModel.NewTile(cdc.tileID, "", session.TileTypeText)
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&tile)
	if err == nil {
		cdc.clientModel.Tiles[tile.ID] = &tile
		cdc.dashboard.Tiles[tile.ID] = true
		cdc.clientModel.UpdateDashboard(&cdc.dashboard)
		err = sess.SaveState()
	}
	sess.WriteError(w, err, http.StatusOK)
}
