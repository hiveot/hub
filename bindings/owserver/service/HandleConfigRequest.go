package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"log/slog"
)

// HandleConfigRequest handles requests to configure the service or devices
func (svc *OWServerBinding) HandleConfigRequest(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	slog.Info("HandleConfigRequest",
		slog.String("thingID", msg.ThingID),
		slog.String("property", msg.Key),
		slog.String("payload", msg.DataAsText()))

	// the thingID is the ROMId of the device to configure
	// the Name is the attributeID of the property to configure
	node, found := svc.nodes[msg.ThingID]
	if !found {
		// unable to delivery to Thing
		err := fmt.Errorf("HandleConfigRequest: Thing '%s' not found", msg.ThingID)
		slog.Warn(err.Error())
		stat.Failed(msg, err)
		return
	}
	attr := node.Attr[msg.Key]
	if !attr.Writable {
		err := fmt.Errorf(
			"HandleConfigRequest: Thing '%s', property '%s' is not writable", msg.ThingID, msg.Key)
		slog.Warn(err.Error())
		stat.Completed(msg, err)
		return
	}
	err := svc.edsAPI.WriteData(msg.ThingID, msg.Key, msg.DataAsText())
	stat.Completed(msg, err)
	return
}
