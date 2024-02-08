package session

// TilePlacement location of the tile in the CSS grid
// breakpoints are set every 100px?
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
	// ID of the dashboard
	ID string
	// Name of the dashboard
	Name string
	// Tiles used in this dashboard
	Tiles []DashboardTile
}

// ClientModel containing all client data
// this is stored in either the browser session store or the state store.
type ClientModel struct {

	// Agent IDs whose events to receive, or all if empty
	Agents []string

	// The client dashboard(s)
	Dashboard []DashboardDefinition
}
