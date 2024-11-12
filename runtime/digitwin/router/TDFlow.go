// Package service with digital twin event handling functions
package router

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"log/slog"
)

// HandlePublishTD agent updates a TD
// This updates the thing TD in the directory
func (svc *DigitwinRouter) HandlePublishTD(agentID string, args any) error {
	slog.Info("HandlePublishTD (from agent)", slog.String("agentID", agentID))

	_, _, err := svc.digitwinAction(
		agentID, digitwin.DirectoryDThingID, digitwin.DirectoryUpdateTDMethod, args, "")
	return err
}

// HandleReadTD consumer reads a TD
func (svc *DigitwinRouter) HandleReadTD(consumerID string, args any) (reply any, err error) {
	_, reply, err = svc.digitwinAction(
		"", digitwin.DirectoryDThingID, digitwin.DirectoryActionReadTD, args, "")
	return reply, err
}

// HandleReadAllTDs consumer reads all TDs
func (svc *DigitwinRouter) HandleReadAllTDs(consumerID string) (reply any, err error) {

	_, reply, err = svc.digitwinAction(
		"", digitwin.DirectoryDThingID, digitwin.DirectoryActionReadAllTDs, nil, "")
	return reply, err
}
