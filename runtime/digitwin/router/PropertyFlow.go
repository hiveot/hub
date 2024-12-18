// Package service with digital twin property handling functions
package router

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"time"
)

// HandleUpdateProperty agent publishes an updated property value to observers.
//
// agentID is the ID of the agent sending the update
// thingID is the ID of the original thing as managed by the agent.
// propName name of the TD defined property
// value property value
func (svc *DigitwinRouter) HandleUpdateProperty(msg *hubclient.ThingMessage) {

	// update the property in the digitwin store and notify observers
	changed, _ := svc.dtwStore.UpdatePropertyValue(
		msg.SenderID, msg.ThingID, msg.Name, msg.Data, msg.CorrelationID)
	if changed {
		dThingID := tdd.MakeDigiTwinThingID(msg.SenderID, msg.ThingID)
		svc.cm.PublishProperty(dThingID, msg.Name, msg.Data, msg.CorrelationID, msg.SenderID)
	}
}

// HandleUpdateMultipleProperties agent publishes a batch with multiple property values.
// This sends individual property updates to observers.
//
// agentID is the ID of the agent sending the update
// thingID is the ID of the original thing as managed by the agent.
// propMap map of property key-values
func (svc *DigitwinRouter) HandleUpdateMultipleProperties(msg *hubclient.ThingMessage) {
	propMap := make(map[string]any)
	err := utils.Decode(msg.Data, &propMap)
	if err != nil {
		slog.Warn("HandleUpdateMultipleProperties: error decoding property map", "err", err.Error())
		return
	}
	// update the property in the digitwin and notify observers for each change
	changes, err := svc.dtwStore.UpdateProperties(
		msg.SenderID, msg.ThingID, propMap, msg.CorrelationID)
	if len(changes) > 0 {
		dThingID := tdd.MakeDigiTwinThingID(msg.SenderID, msg.ThingID)
		for k, v := range changes {
			svc.cm.PublishProperty(dThingID, k, v, msg.CorrelationID, msg.SenderID)
		}
	}
}

// HandleReadProperty consumer requests a digital twin thing's property value
func (svc *DigitwinRouter) HandleReadProperty(
	msg *hubclient.ThingMessage) (stat hubclient.RequestStatus) {

	reply, err := svc.dtwService.ValuesSvc.ReadProperty(msg.SenderID,
		digitwin.ValuesReadPropertyArgs{ThingID: msg.ThingID, Name: msg.Name})

	stat.Completed(msg, reply, err)
	return stat
}

// HandleReadAllProperties consumer requests reading all digital twin's property values
func (svc *DigitwinRouter) HandleReadAllProperties(
	msg *hubclient.ThingMessage) (stat hubclient.RequestStatus) {

	reply, err := svc.dtwService.ValuesSvc.ReadAllProperties(msg.SenderID, msg.ThingID)
	stat.Completed(msg, reply, err)
	return stat
}

// HandleWriteProperty A consumer requests to write a new value to a property.
// The request is forwarded to the Thing and a progress event is sent with the
// progress update of the request.
//
//	write digitwin -> router -> write thing
//	                         -> progress event
//
// if name is empty then newValue contains a map of properties
func (svc *DigitwinRouter) HandleWriteProperty(msg *hubclient.ThingMessage, replyTo string) (
	stat hubclient.RequestStatus) {

	// assign a requestID if none given
	agentID, thingID := tdd.SplitDigiTwinThingID(msg.ThingID)

	// forward the request to the thing's agent and update status
	c := svc.cm.GetConnectionByClientID(agentID)
	if c != nil {
		status, err := c.WriteProperty(thingID, msg.Name, msg.Data, msg.CorrelationID, msg.SenderID)
		if err != nil {
			stat.Failed(msg, err)
		} else {
			stat.Delivered(msg)
			stat.Status = status
		}
	} else {
		stat.Failed(msg, fmt.Errorf("Agent '%s' not reachable", agentID))
	}
	if stat.Status != vocab.RequestCompleted && stat.Status != vocab.RequestFailed {
		// the request is not yet finished. Track it in the active cache.
		svc.mux.Lock()
		svc.activeCache[msg.CorrelationID] = ActionFlowRecord{
			Operation: msg.Operation,
			AgentID:   agentID,
			ThingID:   thingID,
			RequestID: msg.CorrelationID,
			ReplyTo:   replyTo,
			Name:      msg.Name,
			SenderID:  msg.SenderID,
			Progress:  stat.Status,
			Updated:   time.Now(),
		}
		svc.mux.Unlock()
	}
	return stat
}
