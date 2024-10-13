// Package service with digital twin event handling functions
package hubrouter

import (
	"log/slog"
)

// HandleUpdateTDFlow agent updates a TD
func (svc *HubRouter) HandleUpdateTDFlow(agentID string, tdJSON string) error {
	slog.Info("HandleUpdateTDFlow (from agent)", slog.String("agentID", agentID))

	err := svc.dtwService.DirSvc.UpdateDTD(agentID, tdJSON)
	if err != nil {
		slog.Warn("Updating TD failed:", "err", err.Error())
		return err
	}
	return err
}
