package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const RenderSourcesTemplateFile = "RenderSources.gohtml"

type RenderSourcesTemplateData struct {
	Sources []session.TileSource
}

// RenderSources renders the selection of Tile sources
func RenderSources(w http.ResponseWriter, r *http.Request) {

	sess, ctc, err := GetTileContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	data := RenderSourcesTemplateData{}
	//pathArgs := map[string]string{"dashboardID": ctc.dashboardID, "tileID": ctc.tileID}
	//data := RenderTileTemplateData{
	//	Dashboard:          ctc.dashboard,
	//	Tile:               ctc.tile,
	//	RenderEditTilePath: utils.Substitute(RenderEditTilePath, pathArgs),
	//}
	_ = ctc
	buff, err := app.RenderAppOrFragment(r, RenderSourcesTemplateFile, data)
	sess.WritePage(w, buff, err)
}
