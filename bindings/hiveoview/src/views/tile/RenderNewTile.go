package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/teris-io/shortid"
	"net/http"
)

// RenderNewTile renders the Tile editor dialog with a new tile
// If the tile does not exist a new tile will be created
func RenderNewTile(w http.ResponseWriter, r *http.Request) {
	sess, ctc, err := GetTileContext(r, false)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	ctc.tileID = shortid.MustGenerate()
	data := EditTileTemplateData{
		Dashboard:            ctc.dashboard,
		Tile:                 ctc.tile,
		SubmitConfigTilePath: getTilePath(SubmitTilePath, ctc),
	}
	buff, err := app.RenderAppOrFragment(r, EditTileTemplate, data)
	sess.WritePage(w, buff, err)
}
