package tile

import (
	"github.com/hiveot/hub/lib/consumedthing"
	"github.com/hiveot/hub/messaging/tputils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/teris-io/shortid"
	"net/http"
)

const EditTileTemplate = "RenderEditTile.gohtml"

type EditTileTemplateData struct {
	Dashboard session.DashboardModel
	Tile      session.DashboardTile
	// Values of the tile sources by sourceID (affType/thingID/name)
	// FIXME: used consumed thing directory to get values?
	ctDir *consumedthing.ConsumedThingsDirectory
	//Values map[string]*consumedthing.InteractionOutput
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
func (data EditTileTemplateData) GetValue(tileSource session.TileSource) string {
	ct, err := data.ctDir.Consume(tileSource.ThingID)
	if err != nil {
		// should never happen, but just in case
		return err.Error()
	}
	iout := ct.GetValue(tileSource.AffordanceType, tileSource.Name)
	if iout != nil {
		unitSymbol := iout.UnitSymbol()
		return iout.Value.Text() + " " + unitSymbol
	}
	return ""
}
func (data EditTileTemplateData) GetUpdated(tileSource session.TileSource) string {
	ct, err := data.ctDir.Consume(tileSource.ThingID)
	if err != nil {
		return ""
	}
	iout := ct.GetValue(tileSource.AffordanceType, tileSource.Name)
	if iout != nil {
		tputils.DecodeAsDatetime(iout.Updated)
	}
	return ""
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
	ctDir := sess.GetConsumedThingsDirectory()
	data := EditTileTemplateData{
		Dashboard:                   ctc.dashboard,
		Tile:                        ctc.tile,
		TileTypeLabels:              session.TileTypesLabels,
		RenderSelectTileSourcesPath: getTilePath(src.RenderTileSelectSourcesPath, ctc),
		SubmitEditTilePath:          getTilePath(src.PostTileEditPath, ctc),
		ctDir:                       ctDir,
	}
	buff, err := app.RenderAppOrFragment(r, EditTileTemplate, data)
	sess.WritePage(w, buff, err)
}
