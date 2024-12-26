package dashboard

import (
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/services/hiveoview/src/views/tile"
	"github.com/hiveot/hub/transports/tputils"
	"log/slog"
	"net/http"
)

const RenderDashboardTemplate = "RenderDashboardPage.gohtml"

type DashboardPageTemplateData struct {
	Dashboard session.DashboardModel
	// navigation
	DashboardUpdatedEvent            string
	RenderConfirmDeleteDashboardPath string
	RenderConfirmDeleteTilePath      string
	RenderNewTilePath                string
	SubmitDashboardLayoutPath        string
}

// GetTileTemplateData returns empty rendering data for rendering a tile.
// The main purpose is to provide a tile in the dashboard with its hx-get url
func (data DashboardPageTemplateData) GetTileTemplateData(tileID string) tile.RenderTileTemplateData {

	pathArgs := map[string]string{"dashboardID": data.Dashboard.ID, "tileID": tileID}
	renderTilePath := tputils.Substitute(src.RenderTilePath, pathArgs)
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
	slog.Info("RenderDashboardPage",
		slog.String("clientID", sess.GetClientID()),
		slog.String("clcid", sess.GetCLCID()),
		slog.String("remoteAddr", r.RemoteAddr),
	)
	data := DashboardPageTemplateData{}
	data.Dashboard = cdc.dashboard

	// dashboard paths
	data.RenderConfirmDeleteDashboardPath = getDashboardPath(src.RenderDashboardConfirmDeletePath, cdc)
	data.SubmitDashboardLayoutPath = getDashboardPath(src.PostDashboardLayoutPath, cdc)

	// tile paths
	data.RenderNewTilePath = getDashboardPath(src.RenderTileAddPath, cdc)
	data.RenderConfirmDeleteTilePath = getDashboardPath(src.RenderTileConfirmDeletePath, cdc)
	data.DashboardUpdatedEvent = getDashboardPath(src.DashboardUpdatedEvent, cdc)

	// full render or fragment render
	buff, err := app.RenderAppOrFragment(r, RenderDashboardTemplate, data)
	sess.WritePage(w, buff, err)
}
