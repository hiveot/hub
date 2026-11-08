package tile

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/consumer"
	history_client "github.com/hiveot/hivekit/go/cells/history/client"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/services/hiveoview/src/views/history"
)

const RenderTileTemplate = "RenderTile.gohtml"

// Data for RenderTile that renders the single Tile element
type RenderTileTemplateData struct {
	// Tile Title to display
	Tile session.DashboardTile

	// export tile as json
	CopyTileSourcePath string
	// import with the provided definition from clipboard
	PasteTileSourcePath string
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
	sess *session.WebClientSession
	//VM *session.ClientViewModel
	// cts *consumer.ConsumedThingsDirectory
}

// GetHistory returns the 24 hour history for the given thing affordance.
// This truncates the result if there are too many values in the range.
// The max amount of values is the limit set in historyapi.DefaultLimit (1000)
func (dt RenderTileTemplateData) GetHistory(
	thingID string, affType msg.AffordanceType, name string) *history.HistoryTemplateData {

	timestamp := time.Now().Local()
	duration, _ := time.ParseDuration("-24h")
	tdoc := dt.sess.GetTD(thingID)
	hist := history_client.NewReadHistoryClient(dt.sess.GetConsumer())
	values, itemsRemaining, err := hist.ReadHistory(
		thingID, name, timestamp, duration, 500)
	_ = itemsRemaining
	_ = err // ignore for now
	hsd, err := history.NewHistoryTemplateData(
		tdoc, affType, name, values, timestamp, duration)
	_ = err
	return hsd
}

// GetOutputValue return the latest event, property or action output value of a
// tile source, or n/a if not found
//
//	tileSource whose value to display
//
//	This returns the interaction output to display
func (d RenderTileTemplateData) GetOutputValue(tileSource session.TileSource) (iout *consumer.InteractionOutput) {

	// TODO: should this read a value if it isnt found?

	iout = d.sess.GetOutput(tileSource.ThingID, tileSource.AffordanceType, tileSource.Name)
	if iout == nil {
		title := "n/a"
		tdoc := d.sess.GetTD(tileSource.ThingID)
		if tdoc == nil {
			title = "Unknown thing"
		}

		// Thing not found. return a dummy interaction output with a non-schema
		dummy := consumer.InteractionOutput{}
		dummy.AffordanceType = "unknown"
		dummy.ThingID = tileSource.ThingID
		dummy.Name = tileSource.Name
		dummy.Value = msg.ThingValue{Raw: "n/a"}
		dummy.Schema = &wot.DataSchema{Title: title}
		return &dummy
	}
	return iout
}

// GetUnit return the value unit of a tile source
// func (d RenderTileTemplateData) GetUnit(thingID string, name string) string {

// 	cs, err := d.sess.GetConsumedThing(thingID)
// 	if err != nil {
// 		return ""
// 	}
// 	iout := cs.ReadEvent(name)
// 	if iout == nil {
// 		return ""
// 	}
// 	unitSymbol := iout.UnitSymbol()
// 	return unitSymbol
// }

// GetUpdated return the timestamp of a tile source value
// func (d RenderTileTemplateData) GetUpdated(thingID string, name string) string {

// 	cs, err := d.sess.GetConsumedThing(thingID)
// 	if err != nil {
// 		return "n/a"
// 	}
// 	iout := cs.ReadEvent(name)
// 	if iout == nil {
// 		return "n/a"
// 	}
// 	val := utils.FormatDateTime(iout.Timestamp)
// 	return val
// }

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
		CopyTileSourcePath:          getTilePath(src.GetCopyTilePath, ctc),
		PasteTileSourcePath:         getTilePath(src.PostPasteTilePath, ctc),
		RenderEditTilePath:          utils.Substitute(src.RenderTileEditPath, pathArgs),
		RenderConfirmDeleteTilePath: utils.Substitute(src.RenderTileConfirmDeletePath, pathArgs),
		ReRenderTilePath:            utils.Substitute(src.RenderTilePath, pathArgs),
		TileUpdatedEvent:            utils.Substitute(src.TileUpdatedEvent, pathArgs),
		sess:                        sess,
		Locked:                      ctc.dashboard.Locked,
	}
	buff, err := app.RenderAppOrFragment(r, RenderTileTemplate, data)
	sess.WritePage(w, buff, err)
}
