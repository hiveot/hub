package dashboard

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/lib/utils"
	session2 "github.com/hiveot/hub/services/hiveoview/src/session"
	"net/http"
)

type ClientDashboardContext struct {
	clientID    string
	clientModel *session2.ClientDataModel
	dashboardID string
	dashboard   session2.DashboardModel
}

// URL parameters used
const URLParamDashboardID = "dashboardID"

// getDashboardContext is a helper to read session and dashboard from
// the request context.
//
// This reads 'dashboardID' URL parameter, and looks up the corresponding dashboard definition.
// * if no dashboardID is given, then use the first dashboard
// * if createDashboard is set then create a new dashboard if not found
//
//	createDashboard creates a new dashboard if it isn't found (not saved)
func getDashboardContext(r *http.Request, createDashboard bool) (
	*session2.WebClientSession, ClientDashboardContext, error) {

	var found bool
	cdc := ClientDashboardContext{}
	sess, hc, err := session2.GetSessionFromContext(r)
	if err != nil {
		return sess, cdc, err
	}
	cdc.clientID = hc.ClientID()
	cdc.clientModel = sess.GetClientData()
	cdc.dashboardID = chi.URLParam(r, URLParamDashboardID)
	if cdc.dashboardID == "" {
		if len(cdc.clientModel.Dashboards) > 0 {
			dashboard := cdc.clientModel.GetFirstDashboard()
			cdc.dashboardID = dashboard.ID
		} else {
			cdc.dashboardID = "default"
		}
	}
	cdc.dashboard, found = cdc.clientModel.GetDashboard(cdc.dashboardID)
	if !found {
		if createDashboard {
			cdc.dashboard = session2.NewDashboard(cdc.dashboardID, "New Dashboard")
		} else {
			err = fmt.Errorf("Dashboard with ID '%s' not found", cdc.dashboardID)
			return sess, cdc, err
		}
	}
	return sess, cdc, nil
}

// substitute the directory ID in the given path
//
//	 dashboardPath must include the {dashboardID} string
//		cdc dashboard context info
func getDashboardPath(dashboardPath string, cdc ClientDashboardContext) string {
	pathArgs := map[string]string{"dashboardID": cdc.dashboardID}
	return utils.Substitute(dashboardPath, pathArgs)
}
