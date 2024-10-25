// Package service with digital twin event handling functions
package hubrouter

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"log/slog"
)

// HandlePublishTD agent updates a TD
// This updates the thing TD in the directory
func (svc *HubRouter) HandlePublishTD(agentID string, args any) error {
	slog.Info("HandlePublishTD (from agent)", slog.String("agentID", agentID))

	_, _, err := svc.dtwAgent.HandleAction(
		agentID, digitwin.DirectoryDThingID, digitwin.DirectoryUpdateTDMethod, args, "")
	return err
}

// HandleReadTD consumer reads a TD
func (svc *HubRouter) HandleReadTD(consumerID string, args any) (reply any, err error) {
	_, reply, err = svc.dtwAgent.HandleAction(
		"", digitwin.DirectoryDThingID, digitwin.DirectoryActionReadDTD, args, "")
	return reply, err
}

// HandleReadAllTDs consumer reads all TDs
func (svc *HubRouter) HandleReadAllTDs(consumerID string) (reply any, err error) {

	_, reply, err = svc.dtwAgent.HandleAction(
		"", digitwin.DirectoryDThingID, digitwin.DirectoryActionReadAllDTDs, nil, "")
	return reply, err
}
