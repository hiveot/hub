package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
)

// HandleConfigRequest handles requests to configure the service or devices
func (svc *OWServerBinding) HandleConfigRequest(msg *hubclient.ThingMessage) (stat hubclient.ActionProgress) {
	var err error
	valueStr := msg.DataAsText()
	slog.Info("HandleConfigRequest",
		slog.String("thingID", msg.ThingID),
		slog.String("property", msg.Name),
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

	// custom config. Configure the device name and save it in the state service.
	if msg.Name == deviceNameProp {
		// TODO: set device name configuration
	} else {
		attr, found := node.Attr[msg.Name]
		if !found {
			err = fmt.Errorf("HandleConfigRequest: '%s not a property of Thing '%s' found",
				msg.Name, msg.ThingID)
			slog.Warn(err.Error())
		} else if !attr.Writable {
			err := fmt.Errorf(
				"HandleConfigRequest: property '%s' of Thing '%s' is not writable",
				msg.Name, msg.ThingID)
			slog.Warn(err.Error())
		} else {
			err = svc.edsAPI.WriteData(msg.ThingID, msg.Name, valueStr)
		}
	}
	stat.Completed(msg, nil, err)
	return
}
