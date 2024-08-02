package dashboard

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"log/slog"
	"net/http"
)

type ConfirmDeleteTileTemplateData struct {
	Dashboard session.DashboardDefinition
	Tile      session.DashboardTile
}

func RenderConfirmDeleteTile(w http.ResponseWriter, r *http.Request) {
	dashboardID := chi.URLParam(r, "page")
	tileID := chi.URLParam(r, "tileID")

	cs, _, err := session.GetSessionFromContext(r)
	if err != nil {
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}
	clientModel := cs.GetClientData()
	dashboard, found := clientModel.GetDashboard(dashboardID)
	if !found {
		err = fmt.Errorf("Dashboard '%s' not found", dashboardID)
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}
	tile := dashboard.Tiles[tileID]
	data := ConfirmDeleteTileTemplateData{
		Dashboard: dashboard,
		Tile:      tile,
	}
	app.RenderAppOrFragment(w, r, "confirmDeleteTileDialog.gohtml", data)
}

// HandleDeleteTile handles removal of a dashboard tile
func HandleDeleteTile(w http.ResponseWriter, r *http.Request) {
	dashboardID := chi.URLParam(r, "page")
	tileID := chi.URLParam(r, "tileID")

	// get the hub client connection and read the existing TD
	cs, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}

	// delete the tile
	slog.Info("Deleting Tile",
		slog.String("clientID", hc.ClientID()),
		slog.String("dashboardID", dashboardID),
		slog.String("tileID", tileID),
	)
	clientModel := cs.GetClientData()
	dashboard, found := clientModel.GetDashboard(dashboardID)
	if !found {
		err = fmt.Errorf("Dashboard '%s' not found", dashboardID)
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}
	delete(dashboard.Tiles, tileID)

	msgText := fmt.Sprintf("Tile '%s' successfully removed", tileID)
	cs.SendNotify(session.NotifySuccess, msgText)
	w.WriteHeader(http.StatusOK)
}
