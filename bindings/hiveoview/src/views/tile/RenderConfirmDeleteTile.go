package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const ConfirmDeleteTileTemplate = "renderConfirmDeleteTile.gohtml"
const SubmitDeleteTilePath = "/dashboard/{dashboardID}/{tileID}"

type ConfirmDeleteTileTemplateData struct {
	Dashboard            session.DashboardDefinition
	Tile                 session.DashboardTile
	SubmitDeleteTilePath string
}

func RenderConfirmDeleteTile(w http.ResponseWriter, r *http.Request) {

	sess, ctc, err := GetTileContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	data := ConfirmDeleteTileTemplateData{
		Dashboard:            ctc.dashboard,
		Tile:                 ctc.tile,
		SubmitDeleteTilePath: getTilePath(SubmitDeleteTilePath, ctc),
	}
	buff, err := app.RenderAppOrFragment(r, ConfirmDeleteTileTemplate, data)
	sess.WritePage(w, buff, err)
}
