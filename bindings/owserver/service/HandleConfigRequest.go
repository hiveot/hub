package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
)

// HandleConfigRequest handles requests to configure the service or devices
func (svc *OWServerBinding) HandleConfigRequest(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
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
	valueMap := map[string]string{}
	err := msg.Unmarshal(&valueMap)
	if err != nil {
		err := fmt.Errorf("HandleConfigRequest: Invalid properties:", err.Error())
		slog.Warn(err.Error())
		stat.Completed(msg, err)
		return stat
	}
	// keep the last error as a result value
	// iterate all properties
	for key, val := range valueMap {
		attr, found := node.Attr[key]
		if !found {
			err = fmt.Errorf("HandleConfigRequest: '%s not a property of Thing '%s' found",
				key, msg.ThingID)
			slog.Warn(err.Error())
			continue
		} else if !attr.Writable {
			err := fmt.Errorf(
				"HandleConfigRequest: property '%s' of Thing '%s' is not writable", key, msg.ThingID)
			slog.Warn(err.Error())
			continue
		}
		err2 := svc.edsAPI.WriteData(msg.ThingID, key, val)
		if err2 != nil {
			err = err2
		}
	}
	stat.Completed(msg, err)
	return
}
