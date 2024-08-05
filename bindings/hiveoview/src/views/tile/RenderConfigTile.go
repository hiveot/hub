package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const ConfigTileTemplate = "RenderConfigTile.gohtml"
const SubmitConfigTilePath = "/tile/{dashboardID}/{tileID}"

type ConfigTileTemplateData struct {
	Dashboard            session.DashboardDefinition
	Tile                 session.DashboardTile
	SubmitConfigTilePath string
}

// RenderConfigTile renders the Tile editor dialog
// If the tile does not exist a new tile will be created
func RenderConfigTile(w http.ResponseWriter, r *http.Request) {
	cs, ctc, err := getTileContext(r, false)
	if err != nil {
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}
	data := ConfigTileTemplateData{
		Dashboard:            ctc.dashboard,
		Tile:                 ctc.tile,
		SubmitConfigTilePath: getTilePath(SubmitConfigTilePath, ctc),
	}
	app.RenderAppOrFragment(w, r, ConfigTileTemplate, data)
}
