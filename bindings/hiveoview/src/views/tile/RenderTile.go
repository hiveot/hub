package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/consumedthing"
	"net/http"
)

const RenderTileTemplate = "RenderTile.gohtml"
const RenderConfirmDeleteTilePath = "/tile/{dashboardID}/{tileID}/confirmDelete"
const RenderEditTilePath = "/tile/{dashboardID}/{tileID}/edit"
const RenderTilePath = "/tile/{dashboardID}/{tileID}"

// event to notify the dashboard that a redraw is needed
const DashboardUpdatedEvent = "dashboard-updated-{dashboardID}"
const TileUpdatedEvent = "tile-updated-{tileID}"

type RenderTileTemplateData struct {
	// Tile Title to display
	Tile session.DashboardTile

	// path for re-rendering the tile
	ReRenderTilePath string
	// path to rendering edit-tile dialog for menu
	RenderEditTilePath string
	// path to delete time confirmation dialog for menu
	RenderConfirmDeleteTilePath string
	// sse event name to refresh the tile after edit
	TileUpdatedEvent string

	// viewmodel to draw live data from
	//VM *session.ClientViewModel
	cts *consumedthing.ConsumedThingsSession
}

// GetValue return the latest value of a tile source
func (d RenderTileTemplateData) GetValue(thingID string, key string) string {

	cs, err := d.cts.Consume(thingID)
	if err != nil {
		return "n/a"
	}
	iout := cs.ReadEvent(key)
	if iout == nil {
		return "n/a"
	}
	val := iout.ValueAsString() + " " + iout.Schema.UnitSymbol()
	return val
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
		Tile:                        ctc.tile,
		RenderEditTilePath:          utils.Substitute(RenderEditTilePath, pathArgs),
		RenderConfirmDeleteTilePath: utils.Substitute(RenderConfirmDeleteTilePath, pathArgs),
		ReRenderTilePath:            utils.Substitute(RenderTilePath, pathArgs),
		TileUpdatedEvent:            utils.Substitute(TileUpdatedEvent, pathArgs),
		cts:                         sess.GetConsumedThingsSession(),
	}
	buff, err := app.RenderAppOrFragment(r, RenderTileTemplate, data)
	sess.WritePage(w, buff, err)
}
