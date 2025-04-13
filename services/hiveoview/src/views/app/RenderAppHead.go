package app

import (
	"github.com/hiveot/hub/messaging/tputils"
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
	GetDashboardRawPath        string
	RenderAddTilePath          string
	RenderDashboardPath        string
	RenderDeleteDashboardPath  string
	RenderEditDashboardPath    string
	RenderRestoreDashboardPath string
}

// AppHeadTemplateData contains the rendering information for the application header
type AppHeadTemplateData struct {
	Ready                  bool
	Logo                   string
	Title                  string
	Status                 *ConnectStatusTemplateData
	AppHeadDashboards      []AppHeadDashboardData
	ReRenderAppHeadPath    string
	RenderDashboardAddPath string
	RenderAppAboutPath     string
	RenderDirectoryPath    string
	// needed to render the connection status button. must be empty so a fragment re-render is triggered
	RenderConnectStatusPath string
}

// RenderAppHead renders the app header fragment
func RenderAppHead(w http.ResponseWriter, r *http.Request) {

	_, sess, _ := session.GetSessionFromContext(r)
	cm := sess.GetClientData()

	data := AppHeadTemplateData{
		Ready:                  true,
		Logo:                   "/static/hiveot.svg",
		Title:                  "HiveOT",
		Status:                 GetConnectStatus(r),
		AppHeadDashboards:      []AppHeadDashboardData{},
		ReRenderAppHeadPath:    src.RenderAppHeadPath,
		RenderAppAboutPath:     src.RenderAboutPath,
		RenderDirectoryPath:    src.RenderThingDirectoryPath,
		RenderDashboardAddPath: src.RenderDashboardAddPath,
	}

	// add the dashboards from the client data model to the menu
	for _, dashboardModel := range cm.Dashboards {
		pathArgs := map[string]string{"dashboardID": dashboardModel.ID}
		dashboardData := AppHeadDashboardData{
			ID:                         dashboardModel.ID,
			Title:                      dashboardModel.Title,
			GetDashboardRawPath:        tputils.Substitute(src.RenderDashboardExportPath, pathArgs),
			RenderDashboardPath:        tputils.Substitute(src.RenderDashboardPath, pathArgs),
			RenderAddTilePath:          tputils.Substitute(src.RenderTileAddPath, pathArgs),
			RenderDeleteDashboardPath:  tputils.Substitute(src.RenderDashboardDeletePath, pathArgs),
			RenderEditDashboardPath:    tputils.Substitute(src.RenderDashboardEditPath, pathArgs),
			RenderRestoreDashboardPath: tputils.Substitute(src.RenderDashboardImportPath, pathArgs),
		}
		data.AppHeadDashboards = append(data.AppHeadDashboards, dashboardData)
	}
	buff, err := RenderAppOrFragment(r, AppHeadTemplate, data)
	sess.WritePage(w, buff, err)
}
