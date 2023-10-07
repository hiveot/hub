// Package internal handles input set command
package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/plugins/owserver/service/eds"
	"log/slog"
	"time"

	"github.com/hiveot/hub/api/go/vocab"
)

// HandleActionRequest handles requests to activate inputs
func (binding *OWServerBinding) HandleActionRequest(action *hubclient.RequestMessage) error {
	var attr eds.OneWireAttr
	slog.Info("HandleActionRequest",
		slog.String("deviceID", action.DeviceID),
		slog.String("thingID", action.ThingID),
		slog.String("action", action.ActionID),
		slog.String("payload", string(action.Payload)))

	// If the action name is converted to a standardized vocabulary then convert the name
	// to the EDS writable property name.

	// which node is this action for?
	deviceID := action.ThingID
	rawActionID := action.ActionID
	for actID, actType := range eds.ActuatorTypeVocab {
		// check both the 'raw' attribute ID and the vocab ID
		if actID == action.ActionID || actType.ActuatorType == action.ActionID {
			rawActionID = actID
			break
		}
	}

	// TODO: lookup the action name used by the EDS
	edsName := action.ActionID

	// determine the value. Booleans are submitted as integers
	actionValue := action.Payload

	node, found := binding.nodes[deviceID]
	if found {
		attr, found = node.Attr[rawActionID]
	}
	if !found {
		err := fmt.Errorf("action '%s' on unknown attribute '%s'", action.ActionID, attr.Name)
		return err
	} else if !attr.Writable {
		err := fmt.Errorf("action '%s' on read-only attribute '%s'", action.ActionID, attr.Name)
		return err
	}
	// TODO: type conversions needed?
	if attr.DataType == vocab.WoTDataTypeBool {
		//actionValue = fmt.Sprint(ValueAsInt())
	}
	err := binding.edsAPI.WriteData(deviceID, edsName, string(actionValue))

	// read the result
	time.Sleep(time.Second)
	_ = binding.RefreshPropertyValues()

	// Writing the EDS is slow, retry in case it was missed
	time.Sleep(time.Second * 4)
	_ = binding.RefreshPropertyValues()

	if err != nil {
		err = fmt.Errorf("action '%s' failed: %w", action.ActionID, err)
		return err
	}
	_ = action.SendAck()
	return nil
}
