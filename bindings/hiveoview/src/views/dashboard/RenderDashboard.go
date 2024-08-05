package dashboard

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const RenderDashboardTemplate = "RenderDashboard.gohtml"
const RenderNewTilePath = "/tile/{dashboardID}"
const RenderConfirmDeleteDashboardPath = "/dashboard/{dashboardID}/confirmDelete"
const SubmitDashboardLayoutPath = "/dashboard/{dashboardID}/layout"

type DashboardPageTemplateData struct {
	Dashboard                        *session.DashboardDefinition
	Tiles                            []session.DashboardTile
	RenderNewTilePath                string
	RenderConfirmDeleteDashboardPath string
	SubmitDashboardLayoutPath        string
}

// RenderDashboardPage renders the dashboard page or fragment
func RenderDashboardPage(w http.ResponseWriter, r *http.Request) {
	data := DashboardPageTemplateData{}
	cs, cdc, err := getDashboardContext(r, true)
	if err != nil {
		cs.WriteError(w, err, 0)
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
	data.RenderNewTilePath = getDashboardPath(RenderNewTilePath, cdc)
	data.RenderConfirmDeleteDashboardPath = getDashboardPath(RenderConfirmDeleteDashboardPath, cdc)
	data.SubmitDashboardLayoutPath = getDashboardPath(SubmitDashboardLayoutPath, cdc)
	// full render or fragment render
	app.RenderAppOrFragment(w, r, RenderDashboardTemplate, data)
}
