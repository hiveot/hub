package session

// Tile types
const (
	TileTypeText      = "text"
	TileTypeTable     = "table"
	TileTypeAreaChart = "areachart"
	TileTypeBarChart  = "barchart"
	TileTypeLineChart = "linechart"
	TileTypeImage     = "image"
	TileTypeGauge     = "gauge"
)

// TileSource indentifies the thing property/event to display
type TileSource struct {
	ThingID string `json:"thingID"`
	Key     string `json:"key"`
	// optional custom title of the source, overrides TD
	Label string `json:"label,omitempty"`
}

// DashboardTile defines the content of a dashboard tile
type DashboardTile struct {
	// ID of the tile, links to the ID in the layout
	ID string `json:"ID"`
	// Title of the tile
	Title string `json:"title"`
	// Type of tile that controls how it its content is displayed
	// eg Card, Image, AreaChart, BarChart, LineChart,
	TileType string `json:"tileType"`

	// Tile sources
	Sources []TileSource `json:"sources"`
}

type TileLayout struct {
	ID string `json:"id"`
	X  int    `json:"x"`
	Y  int    `json:"y"`
	W  int    `json:"w,omitempty"`
	H  int    `json:"h,omitempty"`
}

// DashboardDefinition containing dashboard properties and its tiles
type DashboardDefinition struct {
	// ID of the dashboard page
	ID string `json:"id"`
	// Title of the dashboard
	Title string `json:"title"`
	// Tiles used in this dashboard
	Tiles map[string]DashboardTile `json:"tiles"`
	// serialized layout
	// eg []{"id":,"x":,"y":,"w":,"h":}
	GridLayout string `json:"layout"`
}

// ClientDataModel containing the persisted client data such dashboard configurations
// this is stored in either the browser session store or the state store.
type ClientDataModel struct {

	// client dashboard(s)
	Dashboards map[string]DashboardDefinition `json:"dashboards"`

	// UI configuration ...
}

// DeleteDashboard removes a dashboard from the model
func (model *ClientDataModel) DeleteDashboard(id string) {
	delete(model.Dashboards, id)
}

// GetDashboard returns a dashboard with the given ID
func (model *ClientDataModel) GetDashboard(id string) (DashboardDefinition, bool) {
	dashboard, found := model.Dashboards[id]
	return dashboard, found
}

// GetFirstDashboard returns the first dashboard in the map
// This returns a new empty dashboard if there are no dashboards
func (model *ClientDataModel) GetFirstDashboard() (d DashboardDefinition) {
	if len(model.Dashboards) == 0 {
		d = model.NewDashboard("default", "New Dashboard")
	}
	for _, d = range model.Dashboards {
		break
	}
	return d
}

func (model *ClientDataModel) NewDashboard(ID string, title string) (d DashboardDefinition) {
	d.ID = "default"
	d.Title = "New Dashboard"
	d.GridLayout = ""
	d.Tiles = make(map[string]DashboardTile)
	return d
}

// UpdateDashboard replaces a dashboard in the model
func (model *ClientDataModel) UpdateDashboard(dashboard *DashboardDefinition) {
	model.Dashboards[dashboard.ID] = *dashboard
}

// UpdateTile replaces a dashboard tile in the model
func (model *ClientDataModel) UpdateTile(dashboardID string, tile DashboardTile) {
	dashboard, found := model.Dashboards[dashboardID]
	if !found {
		dashboard = model.NewDashboard(dashboardID, "New Dashboard")
	}
	dashboard.Tiles[tile.ID] = tile
}
