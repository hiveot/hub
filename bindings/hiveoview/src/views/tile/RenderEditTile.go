package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/things"
	"github.com/teris-io/shortid"
	"net/http"
)

const EditTileTemplate = "RenderEditTile.gohtml"
const RenderSelectTileSourcesPath = "/tile/{dashboardID}/{tileID}/selectSources"
const SubmitTilePath = "/tile/{dashboardID}/{tileID}"

type EditTileTemplateData struct {
	Dashboard session.DashboardModel
	Tile      session.DashboardTile
	// Values of the tile sources by thingID/key
	SourceValues things.ThingMessageMap
	// human labels for each tile type
	TileTypeLabels map[string]string
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
	v, found := data.SourceValues[thingID+"/"+key]
	if !found {
		return ""
	}
	return v.DataAsText()
}
func (data EditTileTemplateData) GetUpdated(thingID, key string) string {
	v, found := data.SourceValues[thingID+"/"+key]
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
	// include the current values of the selected sources
	// the template uses "thingID/key" to obtain the value
	values := make(things.ThingMessageMap)
	for _, tileSource := range ctc.tile.Sources {
		v, err := vm.GetValue(tileSource.ThingID, tileSource.Key)
		if err == nil {
			values[tileSource.ThingID+"/"+tileSource.Key] = v
		}
	}
	data := EditTileTemplateData{
		Dashboard:                   ctc.dashboard,
		Tile:                        ctc.tile,
		TileTypeLabels:              session.TileTypesLabels,
		RenderSelectTileSourcesPath: getTilePath(RenderSelectTileSourcesPath, ctc),
		SubmitEditTilePath:          getTilePath(SubmitTilePath, ctc),
		SourceValues:                values,
	}
	buff, err := app.RenderAppOrFragment(r, EditTileTemplate, data)
	sess.WritePage(w, buff, err)
}
