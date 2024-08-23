package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"log/slog"
	"net/http"
	"strings"
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
	sources, _ := r.Form["sources"]
	sourceTitles, _ := r.Form["sourceTitles"]

	tile, found := cdc.dashboard.GetTile(cdc.tileID)
	if !found {
		// this is a new tile
		tile = cdc.dashboard.NewTile(cdc.tileID, "", session.TileTypeText)
	}
	tile.Title = newTitle
	tile.TileType = tileType
	tile.Sources = make([]session.TileSource, 0)
	// Get the list of sources from the form
	if sources != nil {
		// each source consists of thingID/key
		i := 0
		for _, s := range sources {
			sourceTitle := "?"
			if i < len(sourceTitles) {
				sourceTitle = sourceTitles[i]
			}
			parts := strings.Split(s, "/")
			if len(parts) >= 2 {
				tileSource := session.TileSource{
					ThingID: parts[0],
					Key:     parts[1],
					Title:   sourceTitle,
					// if key had "/" in them then re-join them
					//Title: strings.Join(parts[2:], " / "),
				}
				tile.Sources = append(tile.Sources, tileSource)
			}
			i++
		}
	}

	// add the new tile to the dashboard
	cdc.dashboard.Tiles[tile.ID] = tile
	cdc.clientModel.UpdateDashboard(&cdc.dashboard)
	// save the new dashboard and tile
	err = sess.SaveState()

	if found {
		// Notify the UI that the tile has changed. The eventName was provided
		// in RenderTile.
		eventName := strings.ReplaceAll(TileUpdatedEvent, "{tileID}", tile.ID)
		sess.SendSSE(eventName, "")
	} else {
		// this is a new tile. Notify the dashboard
		eventName := strings.ReplaceAll(DashboardUpdatedEvent, "{dashboardID}", cdc.dashboardID)
		sess.SendSSE(eventName, "")
	}
	sess.WriteError(w, err, http.StatusOK)
}
