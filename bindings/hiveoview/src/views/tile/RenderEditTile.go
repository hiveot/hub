package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/teris-io/shortid"
	"net/http"
)

const EditTileTemplate = "RenderEditTile.gohtml"
const SubmitTilePath = "/tile/{dashboardID}/{tileID}"

type EditTileTemplateData struct {
	Dashboard          session.DashboardModel
	Tile               session.DashboardTile
	SubmitEditTilePath string
	TileTypeLabels     map[string]string
}

func (data EditTileTemplateData) GetTypeLabel(typeID string) string {
	label, found := session.TileTypesLabels[typeID]
	if !found {
		return typeID
	}
	return label
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
	data := EditTileTemplateData{
		Dashboard:          ctc.dashboard,
		Tile:               ctc.tile,
		SubmitEditTilePath: getTilePath(SubmitTilePath, ctc),
		TileTypeLabels:     session.TileTypesLabels,
	}
	buff, err := app.RenderAppOrFragment(r, EditTileTemplate, data)
	sess.WritePage(w, buff, err)
}
