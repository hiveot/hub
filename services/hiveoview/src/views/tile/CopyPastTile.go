package tile

import (
	"net/http"
	"strings"

	"github.com/hiveot/hivehub/services/hiveoview/src"
	"github.com/hiveot/hivehub/services/hiveoview/src/session"
	jsoniter "github.com/json-iterator/go"
)

// CopyTile returns a JSON encoded copy of the tile to the client
// Intended to copy the tile to the clipboard for later pasting
func CopyTile(w http.ResponseWriter, r *http.Request) {
	sess, ctc, err := GetTileContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	tileJson, err := jsoniter.MarshalIndent(ctc.tile, "", "  ")
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// return the tile JSON for the clipboard
	w.Header().Set("Content-Type", "application/json")
	w.Write(tileJson)

	// notify the user with a message
	sess.SendNotify(session.NotifySuccess, "", "Tile copied to the clipboard")
}

// PasteTile replaces the current tile with the given content
// Intended to create a tile from a clipboard copy
func PasteTile(w http.ResponseWriter, r *http.Request) {
	sess, ctc, err := GetTileContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	newTile := session.DashboardTile{}

	// assign a 'new tile' ID
	err = r.ParseForm()
	tileJSON := r.Form.Get("tile")
	err = jsoniter.UnmarshalFromString(tileJSON, &newTile)

	if err != nil {
		sess.SendNotify(session.NotifyWarning, "", "Clipboard does not hold tile data")
	} else if newTile.ID == "" {
		sess.SendNotify(session.NotifyWarning, "", "Clipboard does not hold tile data")
	} else {
		// since the pasted tile has an ID it is assumed to be a valid tile
		newTile.ID = ctc.tile.ID
		ctc.dashboard.UpdateTile(newTile)

		// notify the user with a message
		sess.SendNotify(session.NotifySuccess, "", "Tile updated")
	}
	// notify the UI to refresh the pasted tile
	eventName := strings.ReplaceAll(src.TileUpdatedEvent, "{tileID}", newTile.ID)
	sess.SendSSE(eventName, "")
	w.WriteHeader(http.StatusOK)
}
