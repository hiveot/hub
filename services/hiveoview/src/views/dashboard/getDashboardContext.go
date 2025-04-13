package dashboard

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/messaging/tputils"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"net/http"
)

// URL parameter for dashboard ID
const URLParamDashboardID = "dashboardID"

type ClientDashboardContext struct {
	clientID    string
	clientModel *session.SessionData
	dashboardID string
}

// SelectedDashboard is a convenience function to return the dashboard that is
// selected in the URL.
// This is short for cdc.clientModel.GetDashboard(cdc.dashboardID)
func (cdc *ClientDashboardContext) SelectedDashboard() (session.DashboardModel, bool) {
	d, found := cdc.clientModel.GetDashboard(cdc.dashboardID)
	return d, found
}

// getDashboardContext is a helper to read session and dashboard from
// the request context.
//
// This reads 'dashboardID' URL parameter, and looks up the corresponding dashboard definition.
// * if no dashboardID is given, then use the first dashboard
// * if createDashboard is set then create a new dashboard if not found
//
//	createDashboard creates a new dashboard if it isn't found (not saved)
func getDashboardContext(r *http.Request, createDashboard bool) (
	*session.WebClientSession, ClientDashboardContext, error) {

	cdc := ClientDashboardContext{}
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		return sess, cdc, err
	}
	cdc.clientID = sess.GetClientID()
	cdc.clientModel = sess.GetClientData()
	cdc.dashboardID = chi.URLParam(r, URLParamDashboardID)
	if cdc.dashboardID == "" {
		return sess, cdc, fmt.Errorf("missing dashboard ID")
		//if len(cdc.clientModel.Dashboards) > 0 {
		//	dashboard := cdc.clientModel.GetFirstDashboard()
		//	cdc.dashboardID = dashboard.ID
		//} else {
		//	cdc.dashboardID = "default"
		//}
	}
	dashboard, found := cdc.clientModel.GetDashboard(cdc.dashboardID)
	if !found {
		if createDashboard {
			dashboard = session.NewDashboard(cdc.dashboardID, "New Dashboard")
			// make it available for rendering tiles
			err = cdc.clientModel.UpdateDashboard(&dashboard)
		} else {
			err = fmt.Errorf("Dashboard with ID '%s' not found", cdc.dashboardID)
		}
	}
	return sess, cdc, err
}

// substitute the dashboardID in the given path
//
// dashboardPath must include the {dashboardID} string
// cdc dashboard context info
func getDashboardPath(dashboardPath string, cdc ClientDashboardContext) string {
	pathArgs := map[string]string{"dashboardID": cdc.dashboardID}
	return tputils.Substitute(dashboardPath, pathArgs)
}
