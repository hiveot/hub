package dashboard

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

type EditTileTemplateData struct {
	Dashboard session.DashboardDefinition
	Tile      session.DashboardTile
}

// RenderEditTile renders the Tile editor dialog
func RenderEditTile(w http.ResponseWriter, r *http.Request) {
	data := EditTileTemplateData{}

	cs, _, err := session.GetSessionFromContext(r)
	if err != nil {
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}
	clientModel := cs.GetClientData()
	dashboardID := chi.URLParam(r, "page")
	tileID := chi.URLParam(r, "tileID")
	dashboard, found := clientModel.GetDashboard(dashboardID)
	if !found {
		err = fmt.Errorf("Dashboard '%s' not found", dashboardID)
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}
	tileCfg, found := dashboard.Tiles[tileID]
	if !found {
		err = fmt.Errorf("Tile '%s' not found in dashboard '%s'",
			tileID, dashboardID)
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}
	data.Dashboard = dashboard
	data.Tile = tileCfg
	app.RenderAppOrFragment(w, r, "editTileDialog.gohtml", data)

}
