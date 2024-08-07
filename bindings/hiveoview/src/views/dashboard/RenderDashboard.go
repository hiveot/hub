package dashboard

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const RenderDashboardTemplate = "RenderDashboard.gohtml"
const RenderConfirmDeleteDashboardPath = "/dashboard/{dashboardID}/confirmDelete"
const SubmitDashboardLayoutPath = "/dashboard/{dashboardID}/layout"

// const RenderEditTilePath = "/tile/{dashboardID}/{tileID}/edit"
const RenderNewTilePath = "/tile/{dashboardID}/new"

// const RenderNewTilePath = "/tile/{dashboardID}/{tileID}/edit"
const RenderConfirmDeleteTilePath = "/tile/{dashboardID}/{tileID}/confirmDelete"

type DashboardPageTemplateData struct {
	Dashboard                        *session.DashboardDefinition
	Tiles                            []session.DashboardTile
	RenderConfirmDeleteDashboardPath string
	SubmitDashboardLayoutPath        string

	RenderNewTilePath           string
	RenderConfirmDeleteTilePath string
}

// RenderDashboardPage renders the dashboard page or fragment
func RenderDashboardPage(w http.ResponseWriter, r *http.Request) {
	data := DashboardPageTemplateData{}
	sess, cdc, err := getDashboardContext(r, true)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	data.Dashboard = &cdc.dashboard
	data.Tiles = make([]session.DashboardTile, 0, len(cdc.dashboard.Tiles))
	for tileID, _ := range cdc.dashboard.Tiles {
		tile, found := cdc.clientModel.GetTile(tileID)
		if !found {
			tile = cdc.clientModel.NewTile(tileID, "missing tile", session.TileTypeText)
		}
		data.Tiles = append(data.Tiles, tile)
	}
	// dashboard paths
	data.RenderConfirmDeleteDashboardPath = getDashboardPath(RenderConfirmDeleteDashboardPath, cdc)
	data.SubmitDashboardLayoutPath = getDashboardPath(SubmitDashboardLayoutPath, cdc)

	// tile paths
	data.RenderNewTilePath = getDashboardPath(RenderNewTilePath, cdc)
	data.RenderConfirmDeleteTilePath = getDashboardPath(RenderConfirmDeleteTilePath, cdc)

	// full render or fragment render
	buff, err := app.RenderAppOrFragment(r, RenderDashboardTemplate, data)
	sess.WritePage(w, buff, err)
}
