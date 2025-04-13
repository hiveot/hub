package dashboard

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/messaging/tputils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/services/hiveoview/src/views/tile"
	jsoniter "github.com/json-iterator/go"
	"html/template"
	"log/slog"
	"net/http"
)

const RenderDashboardTemplate = "DashboardPage.gohtml"
const RenderDashboardImportTemplate = "DashboardImport.gohtml"

type DashboardPageTemplateData struct {
	Dashboard session.DashboardModel
	// navigation
	DashboardUpdatedEvent       string
	RenderDeleteDashboardPath   string
	RenderConfirmDeleteTilePath string
	RenderNewTilePath           string
	SubmitDashboardLayoutPath   string
	Background                  template.URL
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
	data.Dashboard, _ = cdc.SelectedDashboard()
	// fixme: redirect in router
	if data.Dashboard.ID == "" {
		http.Redirect(w, r, "/dashboard/default", http.StatusPermanentRedirect)
	}
	// html template requires a 'safe' image source
	data.Background = template.URL(data.Dashboard.Background)

	// dashboard paths
	data.RenderDeleteDashboardPath = getDashboardPath(src.RenderDashboardDeletePath, cdc)
	data.SubmitDashboardLayoutPath = getDashboardPath(src.PostDashboardLayoutPath, cdc)

	// tile paths
	data.RenderNewTilePath = getDashboardPath(src.RenderTileAddPath, cdc)
	data.RenderConfirmDeleteTilePath = getDashboardPath(src.RenderTileConfirmDeletePath, cdc)
	data.DashboardUpdatedEvent = getDashboardPath(src.DashboardUpdatedEvent, cdc)

	// full render or fragment render
	buff, err := app.RenderAppOrFragment(r, RenderDashboardTemplate, data)
	sess.WritePage(w, buff, err)
}

// SubmitDashboardLayout stores the updated dashboard layout.
// Invoked after dragging or resizing tiles in gridstack.
// The layout format is that of gridstack.
//
// @param {dashboardID} with the dashboard ID
//
//	Body is a form containing a field 'layout' with a list of tile placements:
//	"layout": { "id":{tileID}, "x":x,"y":y,"w":w,"h",h}
func SubmitDashboardLayout(w http.ResponseWriter, r *http.Request) {

	sess, cdc, err := getDashboardContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	// Update the layout of the given breakpoint if given
	// the form variables are set through hx-vals at RenderDashboardPage.gohtml:30
	proposedLayoutJSON := r.PostFormValue("layout")
	layoutSize := r.PostFormValue("size")
	if proposedLayoutJSON == "" {
		err = fmt.Errorf("SubmitDashboardLayout: No layout provided")
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// array of {"id":, "x":, "y": "w":, "h":}
	// id must be an existing tile
	var proposedLayout []session.TileLayout
	var newLayout []session.TileLayout
	err = jsoniter.UnmarshalFromString(proposedLayoutJSON, &proposedLayout)
	if err != nil {
		err = fmt.Errorf("SubmitDashboardLayout: Layout is not valid JSON: %s", err.Error())
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// update the dashboard with the new layout
	dashboard, _ := cdc.SelectedDashboard()
	for _, tilePlacement := range proposedLayout {
		_, found := dashboard.Tiles[tilePlacement.ID]
		if found {
			newLayout = append(newLayout, tilePlacement)
		}
	}
	newLayoutSer, _ := json.Marshal(newLayout)

	// avoid unnecessary dashboard layout triggers by grid-stack
	// probable cause is that dashboard tile switch in TextCardInput bubbles event even
	// though hx-post is set.
	existingLayoutSer := dashboard.GridLayouts[layoutSize]
	if string(newLayoutSer) == existingLayoutSer {
		return
	}
	dashboard.GridLayouts[layoutSize] = string(newLayoutSer)

	slog.Info("SubmitDashboardLayout",
		slog.String("dashboardID", cdc.dashboardID),
		slog.String("size", layoutSize))

	// save the updated dashboard
	cdc.clientModel.UpdateDashboard(&dashboard)
	sess.WriteError(w, err, http.StatusOK)
}
