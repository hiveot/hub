// Package internal handles input set command
package service

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/owserver/service/eds"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"log/slog"
	"time"
)

// HandleActionRequest handles requests to activate inputs
func (svc *OWServerBinding) HandleActionRequest(action *things.ThingMessage) (stat api.DeliveryStatus) {
	var attr eds.OneWireAttr
	slog.Info("HandleActionRequest",
		slog.String("thingID", action.ThingID),
		slog.String("key", action.Key),
		slog.String("payload", string(action.Data)))

	if action.Key == vocab.ActionTypeProperties {
		return svc.HandleConfigRequest(action)
	}

	// TODO: lookup the action Title used by the EDS
	edsName := action.Key

	// determine the value. Booleans are submitted as integers
	var actionValue string
	err := json.Unmarshal(action.Data, &actionValue)
	if err != nil {
		stat.Completed(action, err)
		return
	}

	node, found := svc.nodes[action.ThingID]
	if !found {
		err = fmt.Errorf("ID '%s' is not a known node", action.ThingID)
		stat.Completed(action, err)
		return stat
	}
	attr, found = node.Attr[action.Key]
	if !found {
		err = fmt.Errorf("node '%s' found but it doesn't have an action '%s'",
			action.ThingID, action.Key)
		stat.Completed(action, err)
		return stat
	} else if !attr.Writable {
		err = fmt.Errorf("node '%s' action '%s' is a read-only attribute",
			action.ThingID, action.Key)
		stat.Completed(action, err)
		return stat
	}

	// the thingID is the device identifier, eg the ROMId
	err = svc.edsAPI.WriteData(action.ThingID, edsName, string(actionValue))

	// read the result
	time.Sleep(time.Second)
	_ = svc.RefreshPropertyValues()

	// Writing the EDS is slow, retry in case it was missed
	time.Sleep(time.Second * 4)
	_ = svc.RefreshPropertyValues()

	if err != nil {
		err = fmt.Errorf("action '%s' failed: %w", action.Key, err)
	}
	stat.Completed(action, err)
	return stat
}
