// Package internal handles input set command
package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/owserver/service/eds"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
	"time"
)

// HandleActionRequest handles requests to activate inputs
func (svc *OWServerBinding) HandleActionRequest(action *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
	var attr eds.OneWireAttr
	slog.Info("HandleActionRequest",
		slog.String("thingID", action.ThingID),
		slog.String("name", action.Name),
		slog.String("payload", action.DataAsText()),
	)

	if action.MessageType == vocab.MessageTypeProperty {
		return svc.HandleConfigRequest(action)
	}

	// TODO: lookup the action Title used by the EDS
	edsName := action.Name

	node, found := svc.nodes[action.ThingID]
	if !found {
		// delivery failed as the thingID doesn't exist
		err := fmt.Errorf("ID '%s' is not a known node", action.ThingID)
		stat.Failed(action, err)
		return stat
	}
	attr, found = node.Attr[action.Name]
	if !found {
		// delivery completed with error
		err := fmt.Errorf("node '%s' found but it doesn't have an action '%s'",
			action.ThingID, action.Name)
		stat.Completed(action, nil, err)
		return stat
	} else if !attr.Writable {
		// delivery completed with error
		err := fmt.Errorf("node '%s' action '%s' is a read-only attribute",
			action.ThingID, action.Name)
		stat.Completed(action, nil, err)
		return stat
	}

	// Determine the value.
	// FIXME: when building the TD, Booleans are defined as enum integers
	actionValue := action.DataAsText()

	// the thingID is the device identifier, eg the ROMId
	err := svc.edsAPI.WriteData(action.ThingID, edsName, string(actionValue))

	// read the result
	time.Sleep(time.Second)
	_ = svc.RefreshPropertyValues()

	// Writing the EDS is slow, retry in case it was missed
	time.Sleep(time.Second * 4)
	_ = svc.RefreshPropertyValues()

	if err != nil {
		err = fmt.Errorf("action '%s' failed: %w", action.Name, err)
	}
	stat.Completed(action, nil, err)
	return stat
}
