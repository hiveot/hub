package tile

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/consumedthing"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/services/hiveoview/src/views/history"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot/td"
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

// GetOutputValue return the latest event, property or action output value of a
// tile source, or n/a if not found
//
// Tiles also support inputs (actions or property)
func (d RenderTileTemplateData) GetOutputValue(thingID string, name string) (iout *consumedthing.InteractionOutput) {
	ct, _ := d.cts.Consume(thingID)
	if ct == nil {
		// Thing not found. return a dummy interaction output with a non-schema
		tdi := td.NewTD(thingID, "unknown", vocab.ThingDevice)
		dummy := consumedthing.NewInteractionOutput(
			tdi, transports.AffordanceTypeProperty, name, nil, "")
		dummy.Value = consumedthing.NewDataSchemaValue("n/a")
		return dummy
	}
	tdi := ct.GetTD()
	// assume this is an event
	iout = ct.GetEventOutput(name)
	if iout == nil {
		// if not an event get its property. properties might not update immediately
		// so events are preferred.
		iout = ct.GetPropOutput(name)
	}
	// not an event or property, last try is an action output
	if iout == nil {
		aff := tdi.GetAction(name)
		if aff != nil {
			as := ct.QueryAction(name)
			iout = ct.GetActionOutput(as)
		}
	}
	if iout == nil {
		iout = consumedthing.NewInteractionOutput(tdi, "", name,
			"no output value", "")
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
	unitSymbol := iout.UnitSymbol()
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
	val := tputils.DecodeAsDatetime(iout.Updated)
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
		RenderEditTilePath:          tputils.Substitute(src.RenderTileEditPath, pathArgs),
		RenderConfirmDeleteTilePath: tputils.Substitute(src.RenderTileConfirmDeletePath, pathArgs),
		ReRenderTilePath:            tputils.Substitute(src.RenderTilePath, pathArgs),
		TileUpdatedEvent:            tputils.Substitute(src.TileUpdatedEvent, pathArgs),
		cts:                         sess.GetConsumedThingsDirectory(),
	}
	buff, err := app.RenderAppOrFragment(r, RenderTileTemplate, data)
	sess.WritePage(w, buff, err)
}
