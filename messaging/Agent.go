package messaging

import (
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
)

// ThingDirectoryDThingID is the Digitwin ThingID of the runtime Directory Service
// This is duplicated from DirectoryConsumerAPI.go to avoid a compile-time dependency
// on the runtime. (would be circular).
// What would be better is to determine this from discovery but this is a lot
// of work just for cleanliness's sake.
// Used to update the directory with TD's using this agent.
const ThingDirectoryDThingID = "dtw:digitwin:ThingDirectory"
const ThingDirectoryUpdateThingMethod = "updateThing"

// Agent provides the messaging functions needed by hub agents.
// Agents are also consumers as they are able to invoke services.
//
// Hub agents receive requests and return responses.
// The underlying transport protocol binding handles subscription.
type Agent struct {
	*Consumer

	// the application's request handler set with SetRequestHandler
	// intended for sub-protocols that can receive requests. (agents)
	appRequestHandlerPtr atomic.Pointer[RequestHandler]
}

// OnRequest passes a request to the application request handler and returns the response.
// Handler must be set by agent subclasses during init.
// This logs an error if no agent handler is set.
func (ag *Agent) onRequest(
	req *RequestMessage, c IConnection) *ResponseMessage {

	// handle requests if any
	hPtr := ag.appRequestHandlerPtr.Load()
	if hPtr == nil {
		err := fmt.Errorf("Received request but no handler is set")
		resp := req.CreateResponse(nil, err)
		return resp
	}
	resp := (*hPtr)(req, c)
	return resp
}

// PubActionProgress helper for agents to send a 'running' ActionStatus notification
//
// This sends an ActionStatus message with status of running.
func (ag *Agent) PubActionProgress(req RequestMessage, value any) error {
	status := ActionStatus{
		//AgentID:   ag.GetClientID(),
		ActionID:  req.CorrelationID,
		Input:     req.Input,
		Name:      req.Name,
		Output:    value,
		SenderID:  ag.GetClientID(),
		Status:    StatusRunning,
		ThingID:   req.ThingID,
		Requested: req.Created,
		Updated:   utils.FormatNowUTCMilli(),
	}

	resp := NewNotificationMessage(wot.OpInvokeAction, req.ThingID, req.Name, status)
	return ag.cc.SendNotification(resp)
}

// PubEvent helper for agents to send an event to subscribers.
//
// The underlying transport protocol binding handles the subscription mechanism
// as the agent itself doesn't track subscriptions.
func (ag *Agent) PubEvent(thingID string, name string, value any) error {
	// This is a response to subscription request.
	// for now assume this is a hub connection and the hub wants all events
	resp := NewNotificationMessage(wot.OpSubscribeEvent, thingID, name, value)

	return ag.cc.SendNotification(resp)
}

// PubProperty helper for agents to publish a property value notification to observers.
//
// The underlying transport protocol binding handles the subscription mechanism.
func (ag *Agent) PubProperty(thingID string, name string, value any) error {
	// This is a response to an observation request.
	// send the property update as a response to the observe request
	notif := NewNotificationMessage(wot.OpObserveProperty, thingID, name, value)
	slog.Info("PubProperty (async)",
		"thingID", thingID,
		"name", notif.Name,
		"value", notif.ToString(50),
	)
	return ag.cc.SendNotification(notif)
}

// PubProperties helper for agents to publish a map of property values
//
// The underlying transport protocol binding handles the subscription mechanism.
func (ag *Agent) PubProperties(thingID string, propMap map[string]any) error {
	// Implicit rule: if no name is provided the data is a map
	// the transport adds the correlationID of the subscription.
	notif := NewNotificationMessage(wot.OpObserveAllProperties, thingID, "", propMap)

	slog.Info("PubProperties (async)",
		"thingID", thingID,
		"nrProps", len(propMap),
		"value", notif.ToString(50),
	)
	return ag.cc.SendNotification(notif)
}

// SendResponse sends a response for a previous request
func (ag *Agent) SendResponse(resp *ResponseMessage) error {
	return ag.cc.SendResponse(resp)
}

// SetRequestHandler set the application handler for incoming requests
func (ag *Agent) SetRequestHandler(cb RequestHandler) {
	if cb == nil {
		ag.appRequestHandlerPtr.Store(nil)
	} else {
		ag.appRequestHandlerPtr.Store(&cb)
	}
}

// UpdateThing helper for agents to publish an update of a TD in the directory
// Note that this depends on the runtime directory service.
//
// FIXME: change to use directory forms
func (ag *Agent) UpdateThing(tdoc *td.TD) error {
	slog.Info("UpdateThing", slog.String("id", tdoc.ID))

	// TD is sent as JSON
	tdJson, _ := jsoniter.MarshalToString(tdoc)
	err := ag.Rpc(wot.OpInvokeAction, ThingDirectoryDThingID, ThingDirectoryUpdateThingMethod,
		tdJson, nil)
	return err
}

// NewAgent creates a new agent instance for serving requests and sending responses.
// Since agents are also consumers, they can also send requests and receive responses.
//
// Agents can be connected to when running a server or connect to a hub or gateway as client.
//
// This is a wrapper around the ClientConnection that provides WoT response messages
// publishing properties and events to subscribers and publishing a TD.
func NewAgent(cc IConnection,
	connHandler ConnectionHandler,
	notifHandler NotificationHandler,
	reqHandler RequestHandler,
	respHandler ResponseHandler,
	timeout time.Duration) *Agent {

	if timeout == 0 {
		timeout = DefaultRpcTimeout
	}

	//consumer := NewConsumer(cc, respHandler, connHandler, timeout)
	agent := Agent{
		//Consumer: Consumer{
		//	cc:         cc,
		//	rnrChan:    NewRnRChan(),
		//	rpcTimeout: timeout,
		//},
	}
	agent.Consumer = NewConsumer(cc, timeout)
	agent.SetConnectHandler(connHandler)
	agent.SetNotificationHandler(notifHandler)
	agent.SetRequestHandler(reqHandler)
	agent.SetResponseHandler(respHandler)
	//cc.SetNotificationHandler(agent.onNotification)
	//cc.SetResponseHandler(agent.onResponse)
	//cc.SetConnectHandler(agent.onConnect)
	cc.SetRequestHandler(agent.onRequest)
	return &agent
}
