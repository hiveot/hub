package session

import (
	"github.com/google/uuid"
	"sync"
)

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
	// Tiles are the IDs used in this dashboard
	// the value is currently not used
	Tiles map[string]bool `json:"tiles"`
	// Next ID to use
	NextTileID int `json:"nextTileID"`

	// serialized layout
	// eg []{"id":,"x":,"y":,"w":,"h":}
	GridLayout string `json:"layout"`
}

// ClientDataModel containing the persisted client data including
// UI preferences and dashboard configurations.
type ClientDataModel struct {
	mux         sync.RWMutex // mutex to protect the maps below
	dataChanged bool

	// client dashboard(s) - do not use directly
	Dashboards map[string]*DashboardDefinition `json:"dashboards"`
	Tiles      map[string]*DashboardTile       `json:"tiles"`

	// UI preferences ...
}

// Changed returns whether the model has changed
func (model *ClientDataModel) Changed() bool {
	model.mux.RLock()
	defer model.mux.RUnlock()
	return model.dataChanged
}

// DeleteDashboard removes a dashboard from the model
func (model *ClientDataModel) DeleteDashboard(id string) {
	model.mux.Lock()
	defer model.mux.Unlock()

	delete(model.Dashboards, id)
	model.dataChanged = true
}

// GetDashboard returns a dashboard with the given ID
func (model *ClientDataModel) GetDashboard(id string) (d DashboardDefinition, found bool) {
	model.mux.RLock()
	defer model.mux.RUnlock()
	dashboard, found := model.Dashboards[id]
	if found {
		return *dashboard, found
	}
	return
}

// GetFirstDashboard returns the first dashboard in the map
// This returns a new empty dashboard if there are no dashboards
func (model *ClientDataModel) GetFirstDashboard() (d DashboardDefinition) {
	model.mux.RLock()
	defer model.mux.RUnlock()
	if len(model.Dashboards) == 0 {
		d = model.NewDashboard("default", "New Dashboard")
	}
	for _, first := range model.Dashboards {
		d = *first
		break
	}
	return d
}

// NewDashboard create a new dashboard instance with a single default tile.
// Call UpdateDashboard to add it to the model.
func (model *ClientDataModel) NewDashboard(
	ID string, title string) (d DashboardDefinition) {

	if ID == "" {
		ID = uuid.NewString()
	}
	d.ID = "default"
	d.Title = title
	d.GridLayout = ""
	d.Tiles = make(map[string]bool)
	// add a default tile to show. This tile has the dashboard ID
	newTile := model.NewTile(ID+"-tile", "Edit Me", TileTypeText)
	model.Tiles[newTile.ID] = &newTile
	d.Tiles[newTile.ID] = true
	return d
}

// UpdateDashboard adds or replaces a dashboard in the model
func (model *ClientDataModel) UpdateDashboard(dashboard *DashboardDefinition) {
	model.mux.Lock()
	defer model.mux.Unlock()
	model.Dashboards[dashboard.ID] = dashboard
	model.dataChanged = true
}

// GetTile returns a tile in the model
func (model *ClientDataModel) GetTile(id string) (DashboardTile, bool) {
	model.mux.RLock()
	defer model.mux.RUnlock()

	tile, found := model.Tiles[id]
	return *tile, found
}

// NewTile creates a new dashboard tile.
// Call UpdateTile to add it to the model
//
//	 id is the tile ID, or use "" to generate a uuid
//	 title is the title of the Tile
//		tileType is the type of Tile
func (model *ClientDataModel) NewTile(
	id string, title string, tileType string) DashboardTile {
	if id == "" {
		id = uuid.NewString()
	}
	tile := DashboardTile{
		ID:       id,
		Title:    title,
		TileType: tileType,
		Sources:  make([]TileSource, 0),
	}
	return tile
}

// SetChanged sets or clears the 'changed' state of the model.
// Intended to clear it after saving
func (model *ClientDataModel) SetChanged(newValue bool) {
	model.mux.Lock()
	defer model.mux.Unlock()
	model.dataChanged = newValue
}

// UpdateTile adds or replaces a tile in the model
func (model *ClientDataModel) UpdateTile(tile *DashboardTile) {
	model.mux.Lock()
	defer model.mux.Unlock()

	model.Tiles[tile.ID] = tile
	model.dataChanged = true
}

func NewClientDataModel() *ClientDataModel {
	model := ClientDataModel{
		mux:         sync.RWMutex{},
		dataChanged: false,
		Dashboards:  make(map[string]*DashboardDefinition),
		Tiles:       make(map[string]*DashboardTile),
	}
	return &model
}
