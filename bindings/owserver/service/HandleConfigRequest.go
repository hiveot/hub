package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
)

// HandleConfigRequest handles requests to configure the service or devices
func (svc *OWServerBinding) HandleConfigRequest(msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
	var err error
	valueStr := msg.DataAsText()
	slog.Info("HandleConfigRequest",
		slog.String("thingID", msg.ThingID),
		slog.String("property", msg.Key),
		slog.String("payload", valueStr))

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
	attr, found := node.Attr[msg.Key]
	if !found {
		err = fmt.Errorf("HandleConfigRequest: '%s not a property of Thing '%s' found",
			msg.Key, msg.ThingID)
		slog.Warn(err.Error())
	} else if !attr.Writable {
		err := fmt.Errorf(
			"HandleConfigRequest: property '%s' of Thing '%s' is not writable",
			msg.Key, msg.ThingID)
		slog.Warn(err.Error())
	} else {
		err = svc.edsAPI.WriteData(msg.ThingID, msg.Key, valueStr)
	}
	stat.Completed(msg, nil, err)
	return
}
