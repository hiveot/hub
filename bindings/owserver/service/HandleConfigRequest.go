package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
)

// HandleConfigRequest handles requests to configure the service or devices
func (svc *OWServerBinding) HandleConfigRequest(tv *things.ThingMessage) (err error) {
	slog.Info("HandleConfigRequest",
		slog.String("agentID", tv.AgentID),
		slog.String("thingID", tv.ThingID),
		slog.String("property", tv.Name),
		slog.String("payload", string(tv.Data)))

	// the thingID is the ROMId of the device to configure
	// the Name is the attributeID of the property to configure
	node, found := svc.nodes[tv.ThingID]
	if !found {
		err = fmt.Errorf("HandleConfigRequest: Thing '%s' not found", tv.ThingID)
		slog.Warn(err.Error())
		return err
	}
	attr := node.Attr[tv.Name]
	if !attr.Writable {
		err = fmt.Errorf("HandleConfigRequest: Thing '%s', property '%s' is not writable", tv.ThingID, tv.Name)
		slog.Warn(err.Error())
		return err
	}
	err = svc.edsAPI.WriteData(tv.ThingID, tv.Name, string(tv.Data))
	return err
}
