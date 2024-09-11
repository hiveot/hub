package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/wot/consumedthing"
	"github.com/teris-io/shortid"
	"net/http"
)

const EditTileTemplate = "RenderEditTile.gohtml"

type EditTileTemplateData struct {
	Dashboard session.DashboardModel
	Tile      session.DashboardTile
	// Values of the tile sources by thingID/key
	Values map[string]*consumedthing.InteractionOutput
	// human labels for each tile type
	TileTypeLabels map[string]string
	// client view model for everything else
	VM *session.ClientViewModel

	// navigation paths
	RenderSelectTileSourcesPath string // dialog for tile sources selector
	SubmitEditTilePath          string // submit the edited tile
}

func (data EditTileTemplateData) GetTypeLabel(typeID string) string {
	label, found := session.TileTypesLabels[typeID]
	if !found {
		return typeID
	}
	return label
}

// GetValue returns the value of a tile source
func (data EditTileTemplateData) GetValue(thingID, key string) string {
	v, found := data.Values[thingID+"/"+key]
	if !found {
		return ""
	}
	unitSymbol := v.Schema.UnitSymbol()
	return v.Value.Text() + " " + unitSymbol
}
func (data EditTileTemplateData) GetUpdated(thingID, key string) string {
	v, found := data.Values[thingID+"/"+key]
	if !found {
		return ""
	}
	return v.GetUpdated()
}

// RenderEditTile renders the Tile editor dialog
// If the tile does not exist a new tile will be created
func RenderEditTile(w http.ResponseWriter, r *http.Request) {
	sess, ctc, err := GetTileContext(r, false)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// assign a 'new tile' ID if needed
	if ctc.tileID == "" {
		ctc.tileID = shortid.MustGenerate()
	}
	vm := sess.GetViewModel()
	cts := sess.GetConsumedThingsSession()
	// include the current values of the selected sources
	// the template uses "thingID/key" to obtain the value
	values := make(map[string]*consumedthing.InteractionOutput)
	for _, tileSource := range ctc.tile.Sources {
		cs, err := cts.Consume(tileSource.ThingID)
		if err == nil {
			val := cs.ReadEvent(tileSource.Key)
			//v, err := vm.GetValue(tileSource.ThingID, tileSource.Key)
			if val != nil {
				values[tileSource.ThingID+"/"+tileSource.Key] = val
			}
		}
	}
	data := EditTileTemplateData{
		Dashboard:                   ctc.dashboard,
		Tile:                        ctc.tile,
		TileTypeLabels:              session.TileTypesLabels,
		RenderSelectTileSourcesPath: getTilePath(src.RenderTileSelectSourcesPath, ctc),
		SubmitEditTilePath:          getTilePath(src.PostTileEditPath, ctc),
		Values:                      values,
		VM:                          vm,
	}
	buff, err := app.RenderAppOrFragment(r, EditTileTemplate, data)
	sess.WritePage(w, buff, err)
}
