package app

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/lib/utils"
	"net/http"
)

const AppHeadTemplate = "RenderAppHead.gohtml"

// dashboard paths
const RenderAddDashboardPath = "/dashboard/add"
const ReRenderAppHeadPath = "/app/appHead"
const RenderAppAboutPath = "/about"
const RenderDirectoryPath = "/directory"
const RenderConfirmDeleteDashboardPath = "/dashboard/{dashboardID}/confirmDelete"
const RenderDashboardPath = "/dashboard/{dashboardID}"
const RenderEditDashboardPath = "/dashboard/{dashboardID}/config"
const RenderAddTilePath = "/tile/{dashboardID}/add"

//const AppMenuTemplate = "appMenu.gohtml"
//const DashboardMenuTemplate = "dashboardMenu.gohtml"

// AppHeadDashboardData contains the rendering a dashboard menu
type AppHeadDashboardData struct {
	// dashboard title
	ID    string
	Title string
	// paths
	RenderDashboardPath              string
	RenderAddTilePath                string
	RenderConfirmDeleteDashboardPath string
	RenderEditDashboardPath          string
	RenderAddDashboardPath           string
}

// AppHeadTemplateData contains the rendering information for the application header
type AppHeadTemplateData struct {
	Ready               bool
	Logo                string
	Title               string
	Status              *ConnectStatus
	AppHeadDashboards   []AppHeadDashboardData
	ReRenderAppHeadPath string
	RenderAppAboutPath  string
	RenderDirectoryPath string
}

// RenderAppHead renders the app header fragment
func RenderAppHead(w http.ResponseWriter, r *http.Request) {

	sess, _, _ := session.GetSessionFromContext(r)
	cm := sess.GetClientData()

	data := AppHeadTemplateData{
		Ready:               true,
		Logo:                "/static/logo.svg",
		Title:               "HiveOT",
		Status:              GetConnectStatus(r),
		AppHeadDashboards:   []AppHeadDashboardData{},
		ReRenderAppHeadPath: ReRenderAppHeadPath,
		RenderAppAboutPath:  RenderAppAboutPath,
		RenderDirectoryPath: RenderDirectoryPath,
	}

	// add the dashboards from the client data model to the menu
	for _, dashboardModel := range cm.Dashboards {
		pathArgs := map[string]string{"dashboardID": dashboardModel.ID}
		dashboardData := AppHeadDashboardData{
			ID:                               dashboardModel.ID,
			Title:                            dashboardModel.Title,
			RenderDashboardPath:              utils.Substitute(RenderDashboardPath, pathArgs),
			RenderAddTilePath:                utils.Substitute(RenderAddTilePath, pathArgs),
			RenderConfirmDeleteDashboardPath: utils.Substitute(RenderConfirmDeleteDashboardPath, pathArgs),
			RenderEditDashboardPath:          utils.Substitute(RenderEditDashboardPath, pathArgs),
			RenderAddDashboardPath:           utils.Substitute(RenderAddDashboardPath, pathArgs),
		}
		data.AppHeadDashboards = append(data.AppHeadDashboards, dashboardData)
	}
	buff, err := RenderAppOrFragment(r, AppHeadTemplate, data)
	sess.WritePage(w, buff, err)
}
