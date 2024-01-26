package dashboard

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/views"
	"net/http"
)

// TilePlacement location of the tile in the grid
type TilePlacement struct {
	GridX      int
	GridY      int
	GridWidth  int
	GridHeight int
}

// TileIndicator defines the conditions under which an indicator is shown.
//
// The values are stored as a string and are converted based on the property DataSchema
// as defined in the TD. If unknown, a string comparison is used.
type TileIndicator struct {
	// Indicator when value is below
	Below string
	// Indicator when value is above
	Above string
	// Indicator when value is equal
	Equal string
	// type of indicator: exclamation, value
	IndicateAs string
}

// TileSource indentifies the thing property/event to display
type TileSource struct {
	AgentID      string
	ThingID      string
	PropertyName string
	// The indicators to show based on the source value
	Indicator TileIndicator
}

// DashboardTile defines the placement of the tile and its content
type DashboardTile struct {
	// Title of the tile
	Title string
	// Type of tile that controls how it its content is displayed
	// eg Card, Image, AreaChart, BarChart, LineChart,
	Type string
	// Placement of the tile on the dashboard based on the grid
	Placement TilePlacement
	// Tile content
	Sources []TileSource
}

// DashboardDefinition containing dashboard properties and its tiles
type DashboardDefinition struct {
	Name  string
	Tiles []DashboardTile
}

// RenderDashboard renders the dashboard fragment
// This is intended for use from a htmx-get request with a target selector
func RenderDashboard(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]any)
	// when used with htmx, the URL contains the page to display
	pageName := chi.URLParam(r, "page")
	if pageName == "" {
		// when used without htmx there is no page, use the default page
		pageName = "default"
	}
	// TODO: load the dashboard tile configuration for the page name
	// use the session storage
	tiles := make([]DashboardTile, 0)
	data["Dashboard"] = &DashboardDefinition{
		Name:  pageName, // or use the default
		Tiles: tiles,
	}

	views.TM.RenderTemplate(w, r, "dashboard.html", data)
}
