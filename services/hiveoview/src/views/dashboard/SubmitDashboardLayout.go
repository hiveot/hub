package dashboard

import (
	"encoding/json"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"net/http"
)

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
	if proposedLayoutJSON != "" {
		// array of {"id":, "x":, "y": "w":, "h":}
		// id must be an existing tile
		var proposedLayout []session.TileLayout
		var newLayout []session.TileLayout
		err = jsoniter.UnmarshalFromString(proposedLayoutJSON, &proposedLayout)
		if err == nil {
			for _, tilePlacement := range proposedLayout {
				_, found := cdc.dashboard.Tiles[tilePlacement.ID]
				if found {
					newLayout = append(newLayout, tilePlacement)
				}
			}
			newLayoutSer, _ := json.Marshal(newLayout)
			// TODO: fix unnecessary dashboard layout triggers by grid-stack
			// probable cause is that dashboard tile switch in TextCardInput bubbles event even
			// though hx-post is set.
			existingLayoutSer := cdc.dashboard.GridLayouts[layoutSize]
			if string(newLayoutSer) == existingLayoutSer {
				return
			}
			cdc.dashboard.GridLayouts[layoutSize] = string(newLayoutSer)
		}
	}
	if err != nil {
		slog.Warn("SubmitDashboardLayout error",
			slog.String("dashboardID", cdc.dashboard.ID),
			slog.String("size", layoutSize),
			slog.String("err", err.Error()))
		sess.WriteError(w, err, http.StatusBadRequest)
	}
	slog.Info("SubmitDashboardLayout",
		slog.String("dashboardID", cdc.dashboardID),
		slog.String("size", layoutSize))

	// save the updated dashboard
	cdc.clientModel.UpdateDashboard(&cdc.dashboard)
	_ = sess.SaveState()
	sess.WriteError(w, err, http.StatusOK)
}
