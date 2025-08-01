package tile

import (
	"fmt"
	"github.com/hiveot/hub/lib/consumedthing"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils"
	"github.com/hiveot/hub/services/history/historyclient"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/services/hiveoview/src/views/history"
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
	// When the dashboard is locked there is no edit menu
	Locked bool

	// viewmodel to draw live data from
	//VM *session.ClientViewModel
	cts *consumedthing.ConsumedThingsDirectory
}

// GetHistory returns the 24 hour history for the given thing affordance.
// This truncates the result if there are too many values in the range.
// The max amount of values is the limit set in historyapi.DefaultLimit (1000)
func (dt RenderTileTemplateData) GetHistory(
	affType messaging.AffordanceType, thingID string, name string) *history.HistoryTemplateData {
	
	timestamp := time.Now().Local()
	ct, err := dt.cts.Consume(thingID)
	if err != nil {
		return nil
	}
	duration, _ := time.ParseDuration("-24h")
	iout := ct.GetValue(affType, name)
	hist := historyclient.NewReadHistoryClient(ct.GetConsumer())
	values, itemsRemaining, err := hist.ReadHistory(
		iout.ThingID, iout.Name, timestamp, duration, 500)
	_ = itemsRemaining
	_ = err // ignore for now
	hsd, err := history.NewHistoryTemplateData(iout, values, timestamp, duration)
	_ = err
	return hsd
}

// GetOutputValue return the latest event, property or action output value of a
// tile source, or n/a if not found
//
//	tileSource whose value to display
//
//	This returns the interaction output to display
func (d RenderTileTemplateData) GetOutputValue(tileSource session.TileSource) (iout *consumedthing.InteractionOutput) {
	ct, _ := d.cts.Consume(tileSource.ThingID)
	if ct == nil {
		// Thing not found. return a dummy interaction output with a non-schema
		dummy := consumedthing.InteractionOutput{}
		dummy.ThingID = tileSource.ThingID
		dummy.Name = tileSource.Name
		dummy.Value = consumedthing.NewDataSchemaValue("n/a")
		return &dummy
	}

	if tileSource.AffordanceType == messaging.AffordanceTypeAction {
		aff := ct.GetActionAff(tileSource.Name)
		if aff != nil {
			ct.QueryAction(tileSource.Name)
			iout = ct.GetActionOutput(tileSource.Name)
		}
	} else if tileSource.AffordanceType == messaging.AffordanceTypeEvent {
		iout = ct.GetEventOutput(tileSource.Name)
	} else {
		// must be a property
		iout = ct.GetPropertyOutput(tileSource.Name)
	}

	if iout == nil {
		iout = consumedthing.NewInteractionOutput(ct,
			tileSource.AffordanceType, tileSource.Name,
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
	val := utils.FormatDateTime(iout.Timestamp)
	return val
}

// RenderTile renders the single Tile element
// TODO: values from the sources
func RenderTile(w http.ResponseWriter, r *http.Request) {

	sess, ctc, err := GetTileContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	} else if ctc.tile.ID == "" {
		sess.WriteError(w, fmt.Errorf("RenderTile: invalid Tile ID %s", ctc.tileID), http.StatusBadRequest)
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
		Locked:                      ctc.dashboard.Locked,
	}
	buff, err := app.RenderAppOrFragment(r, RenderTileTemplate, data)
	sess.WritePage(w, buff, err)
}
