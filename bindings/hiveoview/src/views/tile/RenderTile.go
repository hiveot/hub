package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const RenderTileTemplate = "RenderTile.gohtml"

type TileTemplateData struct {
	Dashboard session.DashboardDefinition
	// Title to display
	Tile session.DashboardTile
}

// RenderTile renders the single Tile element
// TODO: values from the sources
func RenderTile(w http.ResponseWriter, r *http.Request) {
	data := TileTemplateData{}

	cs, ctc, err := getTileContext(r, true)
	if err != nil {
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}
	data.Dashboard = ctc.dashboard
	data.Tile = ctc.tile
	app.RenderAppOrFragment(w, r, RenderTileTemplate, data)

}
