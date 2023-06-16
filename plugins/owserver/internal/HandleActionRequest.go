// Package internal handles input set command
package internal

import (
	"golang.org/x/exp/slog"
	"time"

	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/plugins/owserver/internal/eds"
)

// HandleActionRequest handles requests to activate inputs
func (binding *OWServerBinding) HandleActionRequest(action *thing.ThingValue) {
	var attr eds.OneWireAttr
	slog.Info("action", "pubID", action.PublisherID, "thingID", action.ThingID, "action", action.ID, "payload", action.Data)

	// If the action name is converted to a standardized vocabulary then convert the name
	// to the EDS writable property name.

	// which node is this action for?
	deviceID := action.ThingID
	rawActionID := action.ID
	for actID, actType := range eds.ActuatorTypeVocab {
		// check both the 'raw' attribute ID and the vocab ID
		if actID == action.ID || actType.ActuatorType == action.ID {
			rawActionID = actID
			break
		}
	}

	// TODO: lookup the action name used by the EDS
	edsName := action.ID

	// determine the value. Booleans are submitted as integers
	actionValue := action.Data

	node, found := binding.nodes[deviceID]
	if found {
		attr, found = node.Attr[rawActionID]
	}
	if !found {
		slog.Warn("action on unknown attribute", "actionID", action.ID, "attrName", attr.Name)
		return
	} else if !attr.Writable {
		slog.Warn("action on read-only attribute", "actionID", action.ID, "attrName", attr.Name)
		return
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
		slog.Warn("action failed", "err", err.Error(), "actionID", action.ID)
	}
}
