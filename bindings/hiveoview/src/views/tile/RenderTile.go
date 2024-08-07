package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const RenderTileTemplate = "RenderTile.gohtml"
const RenderEditTilePath = "/tile/{dashboardID}/{tileID}/edit"

type RenderTileTemplateData struct {
	Dashboard session.DashboardDefinition
	// Title to display
	Tile session.DashboardTile
	//
	RenderEditTilePath string
}

// RenderTile renders the single Tile element
// TODO: values from the sources
func RenderTile(w http.ResponseWriter, r *http.Request) {

	sess, ctc, err := GetTileContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	//pathArgs := map[string]string{"dashboardID": ctc.dashboardID, "tileID": ctc.tileID}
	//data := RenderTileTemplateData{
	//	Dashboard:          ctc.dashboard,
	//	Tile:               ctc.tile,
	//	RenderEditTilePath: utils.Substitute(RenderEditTilePath, pathArgs),
	//}
	buff, err := app.RenderAppOrFragment(r, RenderTileTemplate, ctc.tile)
	sess.WritePage(w, buff, err)
}
