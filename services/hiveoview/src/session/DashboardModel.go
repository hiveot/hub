package session

import (
	"fmt"
	"github.com/hiveot/hub/messaging"
	"github.com/teris-io/shortid"
	"strconv"
)

// Tile types as used in rendering templates
// NOTE: The chart types must match the types in h-timechart.js and EditTile.gohtml
const (
	TileTypeCard         = "card" // table with multiple sources
	TileTypeAreaChart    = "area"
	TileTypeBarChart     = "bar"
	TileTypeLineChart    = "line"
	TileTypeScatterChart = "scatter"
	TileTypeImage        = "image"
	TileTypeRadialGauge  = "radial-gauge"
	TileTypeLinearGauge  = "linear-gauge"
)

var TileTypesLabels = map[string]string{
	TileTypeCard: "Card",
	// charts
	TileTypeLineChart:    "Line Chart",
	TileTypeAreaChart:    "Area Chart",
	TileTypeBarChart:     "Bar Chart",
	TileTypeScatterChart: "Scatter Chart",
	// gauges
	TileTypeRadialGauge: "Radial Gauge",
	TileTypeLinearGauge: "Linear Gauge",
	// other
	TileTypeImage: "Image",
}

// TileSource identifies the thing property/event to display
// this is stored into the dashboard and only contains the info
// necessary to display the list of sources in the tile during edit
// The corresponding TD affordance is provided through a lookup method.
type TileSource struct {
	// Affordance to present "property", "event" or "action"
	AffordanceType messaging.AffordanceType `json:"affordanceType"`
	// ThingID source
	ThingID string `json:"thingID"`
	// Event/property name
	Name string `json:"name"`
	// title of the source, defaults to affordance title
	Title string `json:"title"`
}

// DashboardTile defines the configuration of a dashboard tile
type DashboardTile struct {
	// ID of the tile, links to the ID in the layout
	ID string `json:"ID"`
	// Title of the tile
	Title string `json:"title"`
	// ID of type of tile that controls how it its content is displayed
	// See TileTypeCard, TileType...
	TileType string `json:"tileType"`

	// when type is gauge this offers presentation preset options (thermometer,... )
	// The current gauge type name
	GaugeType string
	// Gauge override of the preset definition (json)
	GaugeOverride string

	// tile background
	BackgroundEnabled      bool   `json:"backgroundEnabled,omitempty"`
	BackgroundColor        string `json:"bgColor,omitempty"`
	BackgroundTransparency string `json:"bgTransparency,omitempty"`
	ImageURL               string `json:"imageURL,omitempty"`
	ImageReloadInterval    int    `json:"imageReloadInterval,omitempty"`

	// Tile sources
	Sources []TileSource `json:"sources"`
}

// RGBA returns the background rgba value of the tile
func (t DashboardTile) GetRGBA() string {
	// color has format #aabbcc
	// rgba has the format rgba(aa,bb,cc, tp)
	tp, _ := strconv.ParseFloat(t.BackgroundTransparency, 10)
	tpInt := int(tp * 255) // to hex
	rgba := fmt.Sprintf("%s%02X", t.BackgroundColor, tpInt)
	return rgba
}

type TileLayout struct {
	ID string `json:"id"`
	X  int    `json:"x"`
	Y  int    `json:"y"`
	W  int    `json:"w,omitempty"`
	H  int    `json:"h,omitempty"`
}

// GridOptions for grid behavior
type GridOptions struct {
	// animate changes to tile reflow on resize/start
	Animate bool `json:"animate"`
	// Float enables fully manual positioning of tiles instead of automatic
	Float bool `json:"float"`
}

// DashboardModel containing dashboard properties and its tiles
type DashboardModel struct {
	// ID of the dashboard page
	ID string `json:"id"`
	// Title of the dashboard
	Title string `json:"title"`

	// Dashboard background image in base64 (if any)
	BackgroundEnabled bool `json:"backgroundEnabled"`
	// uploaded static image
	BackgroundImage string `json:"backgroundImage"` // background image
	// or a URL
	BackgroundURL string `json:"backgroundURL"` // URL
	// auto refresh in seconds
	BackgroundReloadInterval int `json:"backgroundReloadInterval"`
	// sourcefile of static image
	SourceFile string `json:"sourceFile"` // filename

	// time the dashboard was updated
	Updated string `json:"updated"`
	// The dashboard is locked and tiles cannot be edited or moved
	Locked bool `json:"locked"`

	// options for grid behavior
	Grid GridOptions `json:"grid"`

	// Tiles in this dashboard
	Tiles map[string]DashboardTile `json:"tiles"`

	// Serialized layout from gridstack.js per size
	// eg  "sm": []{"id":,"x":,"y":,"w":,"h":}
	GridLayouts map[string]string `json:"layouts"`
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

	d.ID = ID
	d.Title = title
	d.GridLayouts = make(map[string]string)
	d.Tiles = make(map[string]DashboardTile)
	// add a default tile to show. This tile has the dashboard ID
	newTile := d.NewTile(ID+"-tile", "Edit Me", TileTypeCard)
	d.UpdateTile(newTile)
	return d
}
