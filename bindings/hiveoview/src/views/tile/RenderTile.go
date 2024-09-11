package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/history"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/consumedthing"
	"net/http"
	"time"
)

const RenderTileTemplate = "RenderTile.gohtml"

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

// GetHistory returns the 24 hour history for the given key.
// This truncates the result if there are too many values in the range.
// The max amount of values is the limit set in historyapi.DefaultLimit (1000)
func (dt RenderTileTemplateData) GetHistory(thingID string, key string) *history.HistoryTemplateData {
	timestamp := time.Now()
	ct, err := dt.cts.Consume(thingID)

	duration, _ := time.ParseDuration("-24h")
	hsd, err := history.NewHistoryTemplateData(ct, key, timestamp, duration)
	_ = err
	return hsd
}

// GetValue return the latest value of a tile source or nil if not found
// If name is an action then include the action affordance input dataschema
// instead of the event value schema.
func (d RenderTileTemplateData) GetValue(thingID string, name string) (iout *consumedthing.InteractionOutput) {

	cs, err := d.cts.Consume(thingID)
	if err != nil {
		iout.Value = consumedthing.NewDataSchemaValue("n/a")
	}
	iout = cs.ReadEvent(name)
	if iout == nil {
		iout = consumedthing.NewInteractionOutput(thingID, name, nil, nil, "")
		return iout
	}
	// if name is an action then get the input dataschema for allowing
	// direct input of the action from the dashboard tile.
	td := cs.GetThingDescription()
	actionAff := td.GetAction(name)
	if actionAff != nil && actionAff.Input != nil {
		iout.Schema = *actionAff.Input
	}
	return iout
}

// GetUnit return the value unit of a tile source
func (d RenderTileTemplateData) GetUnit(thingID string, key string) string {

	cs, err := d.cts.Consume(thingID)
	if err != nil {
		return ""
	}
	iout := cs.ReadEvent(key)
	if iout == nil {
		return ""
	}
	unitSymbol := iout.Schema.UnitSymbol()
	return unitSymbol
}

// GetUpdated return the timestamp of a tile source value
func (d RenderTileTemplateData) GetUpdated(thingID string, key string) string {

	cs, err := d.cts.Consume(thingID)
	if err != nil {
		return "n/a"
	}
	iout := cs.ReadEvent(key)
	if iout == nil {
		return "n/a"
	}
	val := iout.GetUpdated()
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
		RenderEditTilePath:          utils.Substitute(src.RenderTileEditPath, pathArgs),
		RenderConfirmDeleteTilePath: utils.Substitute(src.RenderTileConfirmDeletePath, pathArgs),
		ReRenderTilePath:            utils.Substitute(src.RenderTilePath, pathArgs),
		TileUpdatedEvent:            utils.Substitute(src.TileUpdatedEvent, pathArgs),
		cts:                         sess.GetConsumedThingsSession(),
	}
	buff, err := app.RenderAppOrFragment(r, RenderTileTemplate, data)
	sess.WritePage(w, buff, err)
}
