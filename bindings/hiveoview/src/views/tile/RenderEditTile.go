package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const EditTileTemplate = "RenderEditTile.gohtml"
const SubmitTilePath = "/tile/{dashboardID}/{tileID}"

type EditTileTemplateData struct {
	Dashboard            session.DashboardDefinition
	Tile                 session.DashboardTile
	SubmitConfigTilePath string
}

// RenderEditTile renders the Tile editor dialog
// If the tile does not exist a new tile will be created
func RenderEditTile(w http.ResponseWriter, r *http.Request) {
	sess, ctc, err := GetTileContext(r, false)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	data := EditTileTemplateData{
		Dashboard:            ctc.dashboard,
		Tile:                 ctc.tile,
		SubmitConfigTilePath: getTilePath(SubmitTilePath, ctc),
	}
	buff, err := app.RenderAppOrFragment(r, EditTileTemplate, data)
	sess.WritePage(w, buff, err)
}
