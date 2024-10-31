package app

import (
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"net/http"
)

const AppHeadTemplate = "RenderAppHead.gohtml"

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

	_, sess, _ := session.GetSessionFromContext(r)
	cm := sess.GetClientData()

	data := AppHeadTemplateData{
		Ready:               true,
		Logo:                "/static/hiveot.svg",
		Title:               "HiveOT",
		Status:              GetConnectStatus(r),
		AppHeadDashboards:   []AppHeadDashboardData{},
		ReRenderAppHeadPath: src.RenderAppHeadPath,
		RenderAppAboutPath:  src.RenderAboutPath,
		RenderDirectoryPath: src.RenderThingDirectoryPath,
	}

	// add the dashboards from the client data model to the menu
	for _, dashboardModel := range cm.Dashboards {
		pathArgs := map[string]string{"dashboardID": dashboardModel.ID}
		dashboardData := AppHeadDashboardData{
			ID:                               dashboardModel.ID,
			Title:                            dashboardModel.Title,
			RenderDashboardPath:              utils.Substitute(src.RenderDashboardPath, pathArgs),
			RenderAddTilePath:                utils.Substitute(src.RenderTileAddPath, pathArgs),
			RenderConfirmDeleteDashboardPath: utils.Substitute(src.RenderDashboardConfirmDeletePath, pathArgs),
			RenderEditDashboardPath:          utils.Substitute(src.PostDashboardConfigPath, pathArgs),
			RenderAddDashboardPath:           utils.Substitute(src.RenderDashboardAddPath, pathArgs),
		}
		data.AppHeadDashboards = append(data.AppHeadDashboards, dashboardData)
	}
	buff, err := RenderAppOrFragment(r, AppHeadTemplate, data)
	sess.WritePage(w, buff, err)
}
