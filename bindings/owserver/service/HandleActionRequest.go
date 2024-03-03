// Package internal handles input set command
package service

import (
	"fmt"
	"github.com/hiveot/hub/bindings/owserver/service/eds"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"time"
)

// HandleActionRequest handles requests to activate inputs
func (svc *OWServerBinding) HandleActionRequest(action *things.ThingValue) (reply []byte, err error) {
	var attr eds.OneWireAttr
	slog.Info("HandleActionRequest",
		slog.String("agentID", action.AgentID),
		slog.String("thingID", action.ThingID),
		slog.String("action", action.Name),
		slog.String("payload", string(action.Data)))

	// TODO: lookup the action Title used by the EDS
	edsName := action.Name

	// determine the value. Booleans are submitted as integers
	actionValue := action.Data

	node, found := svc.nodes[action.ThingID]
	if found {
		attr, found = node.Attr[action.Name]
	}
	if !found {
		err := fmt.Errorf("action '%s' on unknown attribute '%s'",
			action.Name, attr.ID)
		return nil, err
	} else if !attr.Writable {
		err := fmt.Errorf("action '%s' on read-only attribute '%s'",
			action.Name, attr.ID)
		return nil, err
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
		err = fmt.Errorf("action '%s' failed: %w", action.Name, err)
	}
	return nil, err
}
