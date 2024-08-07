package dashboard

import (
	"encoding/json"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"log/slog"
	"net/http"
)

// SubmitDashboardLayout updates the dashboard layout
// @param {dashboardID} with the dashboard ID
// Body is a form containing the new dashboard layout:
//
//	"layout": { "id":{tileID}, "x":x,"y":y,"w":w,"h",h}
func SubmitDashboardLayout(w http.ResponseWriter, r *http.Request) {

	sess, cdc, err := getDashboardContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	slog.Info("SubmitDashboardLayout", "ClientID", cdc.clientID,
		"dashboardID", cdc.dashboardID)

	// Update the layout of the given breakpoint if given
	// TODO: identify breakpoint used: sm, md, lg
	proposedLayoutJSON := r.PostFormValue("layout")
	if proposedLayoutJSON != "" {
		// array of {"id":, "x":, "y": "w":, "h":}
		// id must be an existing tile
		var proposedLayout []session.TileLayout
		var newLayout []session.TileLayout
		err = json.Unmarshal([]byte(proposedLayoutJSON), &proposedLayout)
		if err == nil {
			for _, tilePlacement := range proposedLayout {
				_, found := cdc.dashboard.Tiles[tilePlacement.ID]
				if found {
					newLayout = append(newLayout, tilePlacement)
				}
			}
			newLayoutSer, _ := json.Marshal(newLayout)
			cdc.dashboard.GridLayout = string(newLayoutSer)
		}
	}
	if err != nil {
		slog.Warn("SubmitDashboardLayout error",
			"dashboardID", cdc.dashboard.ID, "err", err.Error())
		sess.WriteError(w, err, http.StatusBadRequest)
	}

	// save the updated dashboard
	cdc.clientModel.UpdateDashboard(&cdc.dashboard)
	_ = sess.SaveState()
	sess.WriteError(w, err, http.StatusOK)
}
