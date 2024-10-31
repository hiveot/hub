package tile

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"net/http"
)

type ClientTileContext struct {
	clientID    string
	clientModel *session.ClientDataModel
	dashboardID string
	dashboard   session.DashboardModel
	tileID      string
	tile        session.DashboardTile
}

// URL parameters used
const URLParamDashboardID = "dashboardID"
const URLParamTileID = "tileID"

// GetTileContext is a helper to read session, dashboard and tile from
// the request context.
//
// This reads 'dashboardID' and 'tileID' URL parameters, and looks up the
// corresponding dashboard and tile definition.
//   - if no dashboardID is given or found, then this fails
//   - if no tileID is given and mustExist is true then this fails
//   - if no tile was found and mustExist is false then a new one is created
func GetTileContext(r *http.Request, mustExist bool) (
	*session.WebClientSession, ClientTileContext, error) {

	var found bool
	ctc := ClientTileContext{}
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		return sess, ctc, err
	}
	ctc.clientID = sess.GetClientID()
	ctc.clientModel = sess.GetClientData()
	ctc.dashboardID = chi.URLParam(r, URLParamDashboardID)
	ctc.dashboard, found = ctc.clientModel.GetDashboard(ctc.dashboardID)
	if !found {
		err = fmt.Errorf("Dashboard with ID '%s' not found", ctc.dashboardID)
		return sess, ctc, err
	}

	ctc.tileID = chi.URLParam(r, URLParamTileID)
	ctc.tile, found = ctc.dashboard.GetTile(ctc.tileID)
	if !found {
		if mustExist {
			err = fmt.Errorf("Tile with ID '%s' not found", ctc.tileID)
			return sess, ctc, err
		}
		ctc.tile = ctc.dashboard.NewTile(ctc.tileID, "New Tile", session.TileTypeText)
	}

	return sess, ctc, nil
}

// substitute the directoryID and tileID in the given path
//
//	 tilePath must include the {dashboardID} and {tileID} strings
//		dashboardID to substitute
//		tileID to substitute
func getTilePath(tilePath string, ctc ClientTileContext) string {
	pathArgs := map[string]string{"dashboardID": ctc.dashboardID, "tileID": ctc.tileID}
	return utils.Substitute(tilePath, pathArgs)
}
