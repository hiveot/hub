package service

import (
	"github.com/hiveot/hub/lib/things"
	"log/slog"
)

// HandleConfigRequest handles requests to configure the service or devices
func (svc *OWServerBinding) HandleConfigRequest(tv *things.ThingValue) (err error) {
	slog.Info("HandleConfigRequest",
		slog.String("agentID", tv.AgentID),
		slog.String("thingID", tv.ThingID),
		slog.String("property", tv.Name),
		slog.String("payload", string(tv.Data)))

	err = svc.edsAPI.WriteData(tv.ThingID, tv.Name, string(tv.Data))
	return err
}
