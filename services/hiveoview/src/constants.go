package src

// shared constants across the service including router paths

// HiveoviewServiceID defines the thingID to access the hiveoview service for
// reading web server status. intended for admin only.
const HiveoviewServiceID = "hiveoview"

// event IDs

const (
	NrActiveSessionsEvent = "activeSessions"

	// event to notify the dashboard that a redraw is needed
	DashboardUpdatedEvent = "dashboard-updated-{dashboardID}"
	TileUpdatedEvent      = "tile-updated-{tileID}"
)

// Router paths
const (
	RenderAboutPath     = "/about"
	RenderAppHeadPath   = "/app/appHead"
	RenderLoginPath     = "/login"
	UIPostLoginPath     = "/login"
	UIPostFormLoginPath = "/loginForm"

	// action paths
	RenderActionRequestPath     = "/action/{thingID}/{name}/request"
	RenderActionStatusPath      = "/status/action/{correlationID}"
	RenderThingPropertyEditPath = "/property/{thingID}/{name}/edit"
	PostActionRequestPath       = "/action/{thingID}/{name}"
	PostThingPropertyEditPath   = "/property/{thingID}/{name}"

	// dashboard paths
	RenderDashboardRootPath    = "/dashboard"
	RenderDashboardDefaultPath = "/dashboard/default"
	RenderDashboardAddPath     = "/dashboard/add"
	RenderDashboardPath        = "/dashboard/{dashboardID}"
	RenderDashboardDeletePath  = "/dashboard/{dashboardID}/delete"
	RenderDashboardEditPath    = "/dashboard/{dashboardID}/config"
	RenderDashboardExportPath  = "/dashboard/{dashboardID}/json"
	RenderDashboardImportPath  = "/dashboard/{dashboardID}/import"
	PostDashboardLayoutPath    = "/dashboard/{dashboardID}/layout"
	PostDashboardEditPath      = "/dashboard/{dashboardID}/edit"
	PostDashboardImportPath    = "/dashboard/{dashboardID}/import"
	PostDashboardDeletePath    = "/dashboard/{dashboardID}/delete"

	// dashboard tile paths
	RenderTileAddPath           = "/tile/{dashboardID}/add"
	RenderTilePath              = "/tile/{dashboardID}/{tileID}"
	RenderTileConfirmDeletePath = "/tile/{dashboardID}/{tileID}/confirmDelete"
	RenderTileEditPath          = "/tile/{dashboardID}/{tileID}/edit"
	RenderTileSelectSourcesPath = "/tile/{dashboardID}/{tileID}/selectSources"
	PostTileEditPath            = "/tile/{dashboardID}/{tileID}"
	PostTileDeletePath          = "/tile/{dashboardID}/{tileID}"

	// directory and related paths
	RenderThingDirectoryPath     = "/directory"
	RenderThingConfirmDeletePath = "/directory/{thingID}/confirmDeleteTD"
	RenderThingDetailsPath       = "/thing/{thingID}/details"
	RenderThingRawPath           = "/thing/{thingID}/raw"
	DeleteThingPath              = "/directory/{thingID}"

	// history paths (duplicated in eventList.gohtml)
	RenderHistoryPagePath           = "/value/{affordanceType}/{thingID}/{name}/history"
	RenderHistoryTimePath           = "/value/{affordanceType}/{thingID}/{name}/history?time="
	RenderHistoryLatestValueRowPath = "/value/{affordanceType}/{thingID}/{name}/latest"

	// other paths
	RenderStatusPath           = "/status"
	RenderConnectionStatusPath = "/status/connection"
)
