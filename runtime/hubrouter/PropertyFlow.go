// Package service with digital twin property handling functions
package hubrouter

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/teris-io/shortid"
	"log/slog"
	"time"
)

// HandlePublishProperty agent publishes an updated property value to observers.
//
// agentID is the ID of the agent sending the update
// thingID is the ID of the original thing as managed by the agent.
// propName in case value is that of a single property
// value either property value or a map of property name-value pairs
func (svc *HubRouter) HandlePublishProperty(
	agentID string, thingID string, propName string, value any, messageID string) (err error) {

	slog.Info("HandlePublishProperty (from agent)",
		slog.String("agentID", agentID),
		slog.String("thingID", thingID),
		slog.String("propName", propName),
		slog.String("messageID", messageID),
		slog.String("value", fmt.Sprintf("%v", value)),
	)
	// update the property in the digitwin and notify observers
	changed, err2 := svc.dtwStore.UpdatePropertyValue(agentID, thingID, propName, value, messageID)
	err = err2
	if changed {
		dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)
		svc.cm.PublishProperty(dThingID, propName, value, messageID, agentID)
	}
	return err
}

// HandleReadProperty handles reading a digital twin thing's property value
func (svc *HubRouter) HandleReadProperty(senderID string, dThingID string, name string) (reply any, err error) {
	reply, err = svc.dtwService.ValuesSvc.ReadProperty(senderID,
		digitwin.ValuesReadPropertyArgs{ThingID: dThingID, Name: name})
	return reply, err
}

// HandleReadAllProperties handles reading all digital twin thing's property values
func (svc *HubRouter) HandleReadAllProperties(senderID string, dThingID string) (reply any, err error) {
	reply, err = svc.dtwService.ValuesSvc.ReadAllProperties(senderID, dThingID)
	return reply, err
}

// HandleWriteProperty A consumer requests to write a new value to a property.
// After authorization, the request is forwarded to the Thing and a progress event
// is sent with the progress update of the request.
//
//	write digitwin -> router -> write thing
//	                         -> progress event
//
// event is sent with the progress update.
// if name is empty then newValue contains a map of properties
func (svc *HubRouter) HandleWriteProperty(
	dThingID string, name string, newValue any, consumerID string) (
	status string, messageID string, err error) {

	slog.Info("HandleWritePropertyFlow (from consumer)",
		slog.String("consumerID", consumerID),
		slog.String("dThingID", dThingID),
		slog.String("name", name),
		slog.String("newValue", fmt.Sprintf("%v", newValue)),
	)
	// assign a messageID if none given
	messageID = "prop-" + shortid.MustGenerate()
	agentID, thingID := tdd.SplitDigiTwinThingID(dThingID)

	// TODO: authorize the request

	// forward the request to the thing's agent and update status
	c := svc.cm.GetConnectionByClientID(agentID)
	status = vocab.ProgressStatusFailed
	if c != nil {
		status, err = c.WriteProperty(thingID, name, newValue, messageID, consumerID)
		//statusInfo = "Thing agent not reachable"
	}
	if status != vocab.ProgressStatusCompleted && status != vocab.ProgressStatusFailed {
		// store incomplete request by message ID to support sending progress updates to the sender
		svc.activeCache[messageID] = &ActionFlowRecord{
			MessageType: vocab.MessageTypeProperty,
			AgentID:     agentID,
			ThingID:     thingID,
			MessageID:   messageID,
			Name:        name,
			SenderID:    consumerID,
			Progress:    status,
			Updated:     time.Now(),
		}
	}
	return status, messageID, err
}
