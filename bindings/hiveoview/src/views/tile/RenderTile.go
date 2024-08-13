package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/utils"
	"net/http"
)

const RenderTileTemplate = "RenderTile.gohtml"
const RenderConfirmDeleteTilePath = "/tile/{dashboardID}/{tileID}/confirmDelete"
const RenderEditTilePath = "/tile/{dashboardID}/{tileID}/edit"
const RenderTilePath = "/tile/{dashboardID}/{tileID}"

type RenderTileTemplateData struct {
	DashboardID string
	// Title to display
	Tile       session.DashboardTile
	TileLabels map[string]string

	// path for re-rendering the tile. Used by the dashboard template.
	//Only used when requiring a call to RenderTile to populate the data.
	ReRenderTilePath string
	// path to rendering edit-tile dialog
	RenderEditTilePath string
	// path to rendering confirmation dialog
	RenderConfirmDeleteTilePath string
}

// RenderTile renders the single Tile element
// TODO: values from the sources
func RenderTile(w http.ResponseWriter, r *http.Request) {

	sess, ctc, err := GetTileContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	pathArgs := map[string]string{"dashboardID": ctc.dashboardID, "tileID": ctc.tileID}
	data := RenderTileTemplateData{
		DashboardID:                 ctc.dashboardID,
		Tile:                        ctc.tile,
		ReRenderTilePath:            "", // no rerender as the data is now available
		RenderEditTilePath:          utils.Substitute(RenderEditTilePath, pathArgs),
		RenderConfirmDeleteTilePath: utils.Substitute(RenderConfirmDeleteTilePath, pathArgs),
	}
	buff, err := app.RenderAppOrFragment(r, RenderTileTemplate, data)
	sess.WritePage(w, buff, err)
}
