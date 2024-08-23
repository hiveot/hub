package dashboard

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/tile"
	"github.com/hiveot/hub/lib/utils"
	"net/http"
)

const RenderDashboardTemplate = "RenderDashboardPage.gohtml"
const RenderConfirmDeleteDashboardPath = "/dashboard/{dashboardID}/confirmDelete"
const SubmitDashboardLayoutPath = "/dashboard/{dashboardID}/layout"

// const RenderEditTilePath = "/tile/{dashboardID}/{tileID}/edit"
const RenderNewTilePath = "/tile/{dashboardID}/new"

// const RenderNewTilePath = "/tile/{dashboardID}/{tileID}/edit"
const RenderConfirmDeleteTilePath = "/tile/{dashboardID}/{tileID}/confirmDelete"

type DashboardPageTemplateData struct {
	Dashboard session.DashboardModel
	// navigation
	RenderConfirmDeleteDashboardPath string
	SubmitDashboardLayoutPath        string
	RenderNewTilePath                string
	RenderConfirmDeleteTilePath      string
	DashboardUpdatedEvent            string
}

// GetTileTemplateData returns empty rendering data for rendering a tile.
// The main purpose is to provide a tile in the dashboard with its hx-get url
func (data DashboardPageTemplateData) GetTileTemplateData(tileID string) tile.RenderTileTemplateData {

	pathArgs := map[string]string{"dashboardID": data.Dashboard.ID, "tileID": tileID}
	renderTilePath := utils.Substitute(tile.RenderTilePath, pathArgs)
	tileTemplateData := tile.RenderTileTemplateData{
		//DashboardID:      data.Dashboard.ID,
		ReRenderTilePath: renderTilePath,
	}
	return tileTemplateData
}

// RenderDashboardPage renders the dashboard page or fragment
func RenderDashboardPage(w http.ResponseWriter, r *http.Request) {
	sess, cdc, err := getDashboardContext(r, true)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	data := DashboardPageTemplateData{}
	data.Dashboard = cdc.dashboard

	// dashboard paths
	data.RenderConfirmDeleteDashboardPath = getDashboardPath(RenderConfirmDeleteDashboardPath, cdc)
	data.SubmitDashboardLayoutPath = getDashboardPath(SubmitDashboardLayoutPath, cdc)

	// tile paths
	data.RenderNewTilePath = getDashboardPath(RenderNewTilePath, cdc)
	data.RenderConfirmDeleteTilePath = getDashboardPath(RenderConfirmDeleteTilePath, cdc)
	data.DashboardUpdatedEvent = getDashboardPath(tile.DashboardUpdatedEvent, cdc)

	// full render or fragment render
	buff, err := app.RenderAppOrFragment(r, RenderDashboardTemplate, data)
	sess.WritePage(w, buff, err)
}
