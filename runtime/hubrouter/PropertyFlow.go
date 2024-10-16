// Package service with digital twin property handling functions
package hubrouter

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/teris-io/shortid"
	"log/slog"
	"time"
)

// HandleUpdatePropertyFlow agent updates the last known thing property value and
// publishes property changes to observers.
// Invoked by agents
//
// agentID is the ID of the agent sending the update
// thingID is the ID of the original thing as managed by the agent.
// propName in case value is that of a single property
// value either property value or a map of property name-value pairs
func (svc *HubRouter) HandleUpdatePropertyFlow(
	agentID string, thingID string, propName string, value any, messageID string) (err error) {

	slog.Info("HandleUpdatePropertyFlow (from agent)",
		slog.String("agentID", agentID),
		slog.String("thingID", thingID),
		slog.String("propName", propName),
		slog.String("messageID", messageID),
		slog.String("value", fmt.Sprintf("%v", value)),
	)
	// probably multiple
	changed, err2 := svc.dtwStore.UpdatePropertyValue(agentID, thingID, propName, value, messageID)
	err = err2
	// FIXME: property changed or last update was X ago
	if changed {
		// notify subscribers of digital twin property changes
		dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)
		// FIXME: publishProperty does not expect a delivery update
		// differentiate from write property in the client
		svc.cm.PublishProperty(dThingID, propName, value, messageID, agentID)
	}
	return err
}

// HandleWritePropertyFlow A consumer requests to write a new value to a property.
// After authorization, the request is forwarded to the Thing and a progress event
// is sent with the progress update of the request.
//
//	write digitwin -> router -> write thing
//	                         -> progress event
//
// event is sent with the progress update.
// if name is empty then newValue contains a map of properties
func (svc *HubRouter) HandleWritePropertyFlow(
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
