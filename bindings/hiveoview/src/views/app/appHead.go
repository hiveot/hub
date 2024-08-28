package app

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/lib/utils"
	"net/http"
)

const AppHeadTemplate = "appHead.gohtml"

// dashboard paths
const RenderAddDashboardPath = "/dashboard/add"
const RenderConfirmDeleteDashboardPath = "/dashboard/{dashboardID}/confirmDelete"
const RenderDashboardPath = "/dashboard/{dashboardID}"
const RenderEditDashboardPath = "/dashboard/{dashboardID}/config"
const RenderNewTilePath = "/tile/{dashboardID}/new"

//const AppMenuTemplate = "appMenu.gohtml"
//const DashboardMenuTemplate = "dashboardMenu.gohtml"

// AppHeadDashboardData contains the rendering a dashboard menu
type AppHeadDashboardData struct {
	// dashboard title
	Title string
	// paths
	RenderDashboardPath              string
	RenderNewTilePath                string
	RenderConfirmDeleteDashboardPath string
	RenderEditDashboardPath          string
	RenderAddDashboardPath           string
}

// AppHeadTemplateData contains the rendering information for the application header
type AppHeadTemplateData struct {
	Ready             bool
	Logo              string
	Title             string
	Status            *ConnectStatus
	AppHeadDashboards []AppHeadDashboardData
}

// RenderAppHead renders the app header fragment
func RenderAppHead(w http.ResponseWriter, r *http.Request) {

	sess, _, _ := session.GetSessionFromContext(r)

	data := AppHeadTemplateData{
		Ready:             true,
		Logo:              "/static/logo.svg",
		Title:             "HiveOT",
		Status:            GetConnectStatus(r),
		AppHeadDashboards: []AppHeadDashboardData{},
	}
	// add the dashboard menus
	dashboardID := "default" // todo: get from client data model
	pathArgs := map[string]string{"dashboardID": dashboardID}
	dashboardData := AppHeadDashboardData{
		Title:                            dashboardID,
		RenderDashboardPath:              utils.Substitute(RenderDashboardPath, pathArgs),
		RenderNewTilePath:                utils.Substitute(RenderNewTilePath, pathArgs),
		RenderConfirmDeleteDashboardPath: utils.Substitute(RenderConfirmDeleteDashboardPath, pathArgs),
		RenderEditDashboardPath:          utils.Substitute(RenderEditDashboardPath, pathArgs),
		RenderAddDashboardPath:           utils.Substitute(RenderAddDashboardPath, pathArgs),
	}
	data.AppHeadDashboards = append(data.AppHeadDashboards, dashboardData)
	buff, err := RenderAppOrFragment(r, AppHeadTemplate, data)
	sess.WritePage(w, buff, err)
}
