package session

import (
	"github.com/teris-io/shortid"
)

// Tile types as used in rendering templates
const (
	TileTypeText      = "text" // table with multiple sources
	TileTypeAreaChart = "areaChart"
	TileTypeBarChart  = "barChart"
	TileTypeLineChart = "lineChart"
	TileTypeImage     = "image"
	TileTypeGauge     = "gauge"
)

var TileTypesLabels = map[string]string{
	TileTypeAreaChart: "Area Chart",
	TileTypeBarChart:  "Bar Chart",
	TileTypeImage:     "Image",
	TileTypeLineChart: "Line Chart",
	TileTypeText:      "Text",
	TileTypeGauge:     "Gauge",
}

// TileSource identifies the thing property/event to display
// this is stored into the dashboard and only contains the info
// necessary to display the list of sources in the tile during edit
// The corresponding TD affordance is provided through a lookup method.
type TileSource struct {
	// ThingID source
	ThingID string `json:"thingID"`
	// Event/property key
	Key string `json:"key"`
	// title of the source, defaults to affordance title
	Title string `json:"title"`
	// source value unit from dataschema
	UnitSymbol string `json:"unitSymbol"`
}

// Affordance returns the event affordance of the tile source
//func (source *TileSource) Affordance() *things.EventAffordance {
//	td := source.TD()
//	aff := td.GetEvent(source.Key)
//	if aff == nil {
//		// return a dummy
//		aff = &things.EventAffordance{
//			Data:  &things.DataSchema{},
//			Title: "not found",
//		}
//	}
//	return aff
//}

// Schema the dataschema of the tile source
//func (source *TileSource) Schema() *things.DataSchema {
//	aff := source.Affordance()
//	ds := aff.Data
//	return ds
//}

// DashboardTile defines the content of a dashboard tile
type DashboardTile struct {
	// ID of the tile, links to the ID in the layout
	ID string `json:"ID"`
	// Title of the tile
	Title string `json:"title"`
	// ID of type of tile that controls how it its content is displayed
	// See TileTypeText, TileType...
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

// DashboardModel containing dashboard properties and its tiles
type DashboardModel struct {
	// ID of the dashboard page
	ID string `json:"id"`
	// Title of the dashboard
	Title string `json:"title"`
	// Tiles in this dashboard
	Tiles map[string]DashboardTile `json:"tiles"`

	// Next ID to use
	NextTileID int `json:"nextTileID"`

	// serialized layout
	// eg []{"id":,"x":,"y":,"w":,"h":}
	GridLayout string `json:"layout"`
}

// GetTile returns the dashboard tile with the given ID
// This returns a new tile if id is empty or doesn't exist
func (d *DashboardModel) GetTile(tileID string) (t DashboardTile, found bool) {
	if tileID == "" {
		tileID = shortid.MustGenerate()
	}
	t, found = d.Tiles[tileID]
	if !found {
		t.ID = tileID
		t.Title = "Tile not found"
	}
	return t, found
}

// NewTile creates a new dashboard tile.
//
// The tile is not added until UpdateTile is invoked.
//
//	id is the tile ID, or use "" to generate a short-id
//	title is the title of the Tile
//	tileType is the type of Tile
func (d *DashboardModel) NewTile(
	id string, title string, tileType string) DashboardTile {
	if id == "" {
		id = shortid.MustGenerate()
	}
	tile := DashboardTile{
		ID:       id,
		Title:    title,
		TileType: tileType,
		Sources:  make([]TileSource, 0),
	}
	return tile
}

func (d *DashboardModel) UpdateTile(t DashboardTile) {
	d.Tiles[t.ID] = t
}

// NewDashboard create a new dashboard instance with a single default tile.
// Call UpdateDashboard to add it to the model.
func NewDashboard(
	ID string, title string) (d DashboardModel) {

	if ID == "" {
		ID = shortid.MustGenerate()
	}
	d.ID = "default"
	d.Title = title
	d.GridLayout = ""
	d.Tiles = make(map[string]DashboardTile)
	// add a default tile to show. This tile has the dashboard ID
	newTile := d.NewTile(ID+"-tile", "Edit Me", TileTypeText)
	d.UpdateTile(newTile)
	return d
}