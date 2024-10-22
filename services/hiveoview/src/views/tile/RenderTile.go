package tile

import (
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/services/hiveoview/src/views/history"
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
	cts *consumedthing.ConsumedThingsDirectory
}

// GetHistory returns the 24 hour history for the given key.
// This truncates the result if there are too many values in the range.
// The max amount of values is the limit set in historyapi.DefaultLimit (1000)
func (dt RenderTileTemplateData) GetHistory(thingID string, name string) *history.HistoryTemplateData {
	timestamp := time.Now()
	ct, err := dt.cts.Consume(thingID)
	if err != nil {
		return nil
	}
	duration, _ := time.ParseDuration("-24h")
	hsd, err := history.NewHistoryTemplateData(ct, name, timestamp, duration)
	_ = err
	return hsd
}

// GetValue return the latest value of a tile source or nil if not found
//
//  1. If name is an event then use the latest event value. (read-only)
//  2. If name is not an event but a property then use the property value. (read/write)
//  3. If name is an action with an input schema then use this schema with the event/prop value
//     what if the event/prop value type differs from action input type?
func (d RenderTileTemplateData) GetValue(thingID string, name string) (iout *consumedthing.InteractionOutput) {
	var rawValue any
	var updated string
	ct, _ := d.cts.Consume(thingID)
	if ct == nil {
		// Thing not found. return a dummy interaction output with a non-schema
		iout = consumedthing.NewInteractionOutput(
			thingID, name, nil, nil, "")
		iout.Value = consumedthing.NewDataSchemaValue("n/a")
		return iout
	}
	td := ct.GetThingDescription()
	// assume this is an event
	iout = ct.ReadEvent(name)
	if iout == nil {
		// if not an event get its property. properties might not update immediately
		// so events are preferred.
		iout = ct.ReadProperty(name)
	}
	// obtain the value to display from either event or property
	if iout != nil {
		rawValue = iout.Value.Raw()
		updated = iout.Updated
	}

	// if name is also an action with an input schema, then get this schema with the event/prop value
	actionAff := td.GetAction(name)
	if actionAff != nil && actionAff.Input != nil {
		if iout != nil && iout.Schema.Type != actionAff.Input.Type {
			// FIXME: rawValue must be of the same type as the action input otherwise
			//  it might not display correctly. Lets just roll with this for now
			//  until a recovery solution is known.
		}
		iout = consumedthing.NewInteractionOutput(
			thingID, name, actionAff.Input, rawValue, updated)
	}
	return iout
}

// GetUnit return the value unit of a tile source
func (d RenderTileTemplateData) GetUnit(thingID string, name string) string {

	cs, err := d.cts.Consume(thingID)
	if err != nil {
		return ""
	}
	iout := cs.ReadEvent(name)
	if iout == nil {
		return ""
	}
	unitSymbol := iout.Schema.UnitSymbol()
	return unitSymbol
}

// GetUpdated return the timestamp of a tile source value
func (d RenderTileTemplateData) GetUpdated(thingID string, name string) string {

	cs, err := d.cts.Consume(thingID)
	if err != nil {
		return "n/a"
	}
	iout := cs.ReadEvent(name)
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
		cts:                         sess.GetConsumedThingsDirectory(),
	}
	buff, err := app.RenderAppOrFragment(r, RenderTileTemplate, data)
	sess.WritePage(w, buff, err)
}
