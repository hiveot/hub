package dashboard

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"io"
	"log/slog"
	"net/http"
)

const DashboardTemplate = "dashboardPage.gohtml"

type DashboardTemplateData struct {
	Dashboard *session.DashboardDefinition
}

// DeleteDashboard removes the dashboard
// @param {page} with the dashboard ID
func DeleteDashboard(w http.ResponseWriter, r *http.Request) {
	cs, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		cs.WriteError(w, err, 0)
		return
	}
	_ = hc
	dashboardID := chi.URLParam(r, "page")
	if dashboardID == "" {
		err = fmt.Errorf("Missing dashboard ID")
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}
	clientModel := cs.GetClientData()
	clientModel.DeleteDashboard(dashboardID)
	err = cs.SaveState()
	cs.WriteError(w, err, http.StatusOK)
}

// DeleteTile removes a tile from the dashboard
// @param {page} with the dashboard ID
// @param {tileID} with the ID of the tile to remove
func DeleteTile(w http.ResponseWriter, r *http.Request) {
	cs, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		cs.WriteError(w, err, 0)
		return
	}
	_ = hc
	dashboardID := chi.URLParam(r, "page")
	tileID := chi.URLParam(r, "tileID")
	if dashboardID == "" || tileID == "" {
		err = fmt.Errorf("Missing dashboard or tile ID")
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}
	clientModel := cs.GetClientData()
	dashboard, found := clientModel.Dashboards[dashboardID]
	if !found {
		err = fmt.Errorf("Dashboard '%s' not found", dashboardID)
		cs.WriteError(w, err, http.StatusBadRequest)
	}
	_, found = dashboard.Tiles[tileID]
	if !found {
		err = fmt.Errorf("Tile '%s' not found in dashboard '%s'",
			tileID, dashboardID)
		cs.WriteError(w, err, http.StatusBadRequest)
	}
	delete(dashboard.Tiles, tileID)
	clientModel.UpdateDashboard(&dashboard)
	err = cs.SaveState()
	cs.WriteError(w, err, http.StatusOK)
}

//// Request to load the dashboards of the given client
//func LoadClientDashboards(w http.ResponseWriter, r *http.Request) {
//
//}
//
//// SaveClientDashboards persists the client dashboards
//// body contains the json encoded dashboards
//func SaveClientDashboards(w http.ResponseWriter, r *http.Request) {
//	//data, err := io.ReadAll(r.Body)
//
//}

// RenderDashboard renders the dashboard page or fragment
// This is intended for use from a htmx-get request with a target selector
func RenderDashboard(w http.ResponseWriter, r *http.Request) {
	data := DashboardTemplateData{}

	cs, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		cs.WriteError(w, err, 0)
		return
	}
	clientModel := cs.GetClientData()
	_ = hc
	// the URL contains the page to display. If not provided then return
	// the default dashboard.
	dashboardID := chi.URLParam(r, "page")
	if dashboardID == "" {
		// if no page is given return the first dashboard
		if len(clientModel.Dashboards) > 0 {
			dashboard := clientModel.GetFirstDashboard()
			dashboardID = dashboard.ID
		} else {
			// when no dashboards are defined use 'default'
			dashboardID = "default"
		}
	}
	dashboard, found := clientModel.GetDashboard(dashboardID)
	if !found {
		dashboard = clientModel.NewDashboard(dashboardID, "New Dashboard")
	}

	if len(dashboard.Tiles) == 0 {
		// add a default tile to show
		newTile := session.DashboardTile{
			ID:       "firstTile",
			Title:    "new tile",
			TileType: session.TileTypeText,
			Sources:  nil,
		}
		dashboard.Tiles = map[string]session.DashboardTile{newTile.ID: newTile}
		clientModel.UpdateDashboard(&dashboard)
	}
	data.Dashboard = &dashboard

	// full render or fragment render
	app.RenderAppOrFragment(w, r, DashboardTemplate, data)
}

// HandleUpdateDashboard updates the dashboard name and layout
// @param {page} with the dashboard ID
// Body is a json document containing the changes to make:
//
//	"title" when to change the title
//	"layout" with a list of tile IDs and positions:
//	   { "id":{tileID}, "x":x,"y":y,"w":w,"h",h}
func HandleUpdateDashboard(w http.ResponseWriter, r *http.Request) {
	cs, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		cs.WriteError(w, err, 0)
		return
	}
	_ = hc
	clientModel := cs.GetClientData()

	// a dashboard page ID is required
	dashboardID := chi.URLParam(r, "page")

	slog.Info("Saving dashboard", "ClientID", hc.ClientID(),
		"ID", dashboardID)
	dashboard, found := clientModel.GetDashboard(dashboardID)
	if !found {
		dashboard = clientModel.NewDashboard(dashboardID, "New Dashboard")
	}

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
				_, found := dashboard.Tiles[tilePlacement.ID]
				if found {
					newLayout = append(newLayout, tilePlacement)
				}
			}
			newLayoutSer, _ := json.Marshal(newLayout)
			dashboard.GridLayout = string(newLayoutSer)
		}
	}
	if err == nil {
		cs.WriteError(w, err, http.StatusBadRequest)
	}
	// update the dashboard title if given
	newTitle := r.PostFormValue("title")
	if newTitle != "" {
		dashboard.Title = newTitle
	}
	// save the updated dashboard
	clientModel.UpdateDashboard(&dashboard)
	cs.SetClientData(clientModel)
	err = cs.SaveState()
	cs.WriteError(w, err, http.StatusOK)
}

// HandleUpdateTile updates a dashboard tile configuration
// @param {page} with the dashboard ID
// @param {tileID} with the tile ID
//
// Body is a json document containing the tile configuration
func HandleUpdateTile(w http.ResponseWriter, r *http.Request) {
	cs, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		cs.WriteError(w, err, 0)
		return
	}
	_ = hc
	clientModel := cs.GetClientData()

	// a dashboard page ID is required
	dashboardID := chi.URLParam(r, "page")
	tileID := chi.URLParam(r, "tileID")

	slog.Info("Updating dashboard tile",
		"ClientID", hc.ClientID(),
		"dashboardID", dashboardID,
		"tileID", tileID)

	if dashboardID == "" || tileID == "" {
		err = fmt.Errorf("Missing dashboard or tile ID")
		cs.WriteError(w, err, http.StatusBadRequest)
		return
	}
	dashboard, found := clientModel.GetDashboard(dashboardID)
	if !found {
		err = fmt.Errorf("Dashboard '%s' not found", dashboardID)
		cs.WriteError(w, err, http.StatusBadRequest)
	}
	tile, found := dashboard.Tiles[tileID]
	if !found {
		tile = session.DashboardTile{
			ID: tileID, Title: "New Tile", TileType: session.TileTypeText}
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		cs.WriteError(w, err, http.StatusBadRequest)
	}
	// replace updated fields
	err = json.Unmarshal(data, &tile)
	if err != nil {
		cs.WriteError(w, err, http.StatusBadRequest)
	}
	clientModel.UpdateTile(dashboardID, tile)
	clientModel.UpdateDashboard(&dashboard)
	err = cs.SaveState()
	cs.WriteError(w, err, http.StatusOK)
}
