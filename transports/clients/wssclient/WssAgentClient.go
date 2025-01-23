package wssclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/wssserver"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"time"
)

// WssAgentTransport manages the agent connection to the hub server using Websockets.
// This implements the IAgentTransport interface.
type WssAgentTransport struct {
	WssConsumerClient
	// the application's request handler set with SetRequestHandler
	// intended for sub-protocols that can receive requests.
	AppRequestHandler transports.RequestHandler
}

// handle agent requests if any
func (cl *WssAgentTransport) handleAgentRequest(req transports.RequestMessage) {
	if cl.AppRequestHandler == nil {
		slog.Error("handleAgentRequest: no request handler set",
			"clientID", cl.GetClientID(),
			"op", req.Operation,
		)
		return
	}
	resp := cl.AppRequestHandler(req)

	// send the response to the caller
	err := cl.SendResponse(resp)
	if err != nil {
		slog.Error("handleAgentRequest: failed", "err", err.Error())
	}
}

// PubEvent helper for agents to publish an event
// This is short for SendNotification( ... wot.OpEvent ...)
func (cl *WssAgentTransport) PubEvent(thingID string, name string, value any) error {
	notif := transports.NewNotificationResponse(wot.HTOpEvent, thingID, name, value)
	return cl.SendNotification(notif)
}

// PubProperty helper for agents to publish a property value update
// This is short for SendNotification( ... wot.OpProperty ...)
func (cl *WssAgentTransport) PubProperty(thingID string, name string, value any) error {

	notif := transports.NewNotificationResponse(wot.HTOpUpdateProperty, thingID, name, value)
	return cl.SendNotification(notif)
}

// PubProperties helper for agents to publish a map of property values
func (cl *WssAgentTransport) PubProperties(thingID string, propMap map[string]any) error {

	notif := transports.NewNotificationResponse(wot.HTOpUpdateMultipleProperties, thingID, "", propMap)
	err := cl.SendNotification(notif)
	return err
}

// PubTD helper for agents to publish a TD update
// This is short for SendNotification( ... wot.HTOpTD ...)
func (cl *WssAgentTransport) PubTD(td *td.TD) error {
	tdJson, _ := jsoniter.Marshal(td)
	notif := transports.NewNotificationResponse(wot.HTOpUpdateTD, td.ID, "", tdJson)
	return cl.SendNotification(notif)
}

// wssToRequest converts a websocket message to the unified request message
func (cl *WssAgentTransport) wssToRequest(
	baseMsg wssserver_old.BaseMessage, raw []byte) (isRequest bool, req transports.RequestMessage) {

	var err error
	isRequest = true

	msgType := baseMsg.MessageType
	correlationID := baseMsg.CorrelationID
	slog.Info("WssClientHandleMessage",
		slog.String("clientID", cl.GetClientID()),
		slog.String("msgType", msgType),
		slog.String("correlationID", correlationID),
	)
	operation, _ := wssserver_old.MsgTypeToOp[baseMsg.MessageType]

	switch baseMsg.MessageType {

	// agent receives request messages
	case wssserver_old.MsgTypeInvokeAction,
		wssserver_old.MsgTypeQueryAction,
		wssserver_old.MsgTypeQueryAllActions:
		wssMsg := wssserver_old.ActionMessage{}
		err = cl.Unmarshal(raw, &wssMsg)
		req = transports.NewRequestMessage(
			wot.OpInvokeAction, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, wssMsg.CorrelationID)
		req.Created = wssMsg.Timestamp
		req.SenderID = wssMsg.SenderID

	// agent receivess read event or property requests
	case wssserver_old.MsgTypeReadEvent, wssserver_old.MsgTypeReadAllEvents,
		wssserver_old.MsgTypeReadProperty, wssserver_old.MsgTypeReadAllProperties,
		wssserver_old.MsgTypeReadMultipleProperties,
		wssserver_old.MsgTypeReadTD:

		wssMsg := wssserver_old.EventMessage{}
		err = cl.Unmarshal(raw, &wssMsg)
		req = transports.NewRequestMessage(
			operation, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, wssMsg.CorrelationID)
		req.Created = wssMsg.Timestamp

	case wssserver_old.MsgTypeWriteProperty, wssserver_old.MsgTypeWriteMultipleProperties:
		wssMsg := wssserver_old.PropertyMessage{}
		err = cl.Unmarshal(raw, &wssMsg)
		req = transports.NewRequestMessage(
			operation, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, wssMsg.CorrelationID)
		req.Created = wssMsg.Timestamp
		req.SenderID = wssMsg.SenderID

	default:
		isRequest = false
	}
	if err != nil {
		isRequest = false
	}
	return isRequest, req
}

// HandleAgentMessage agent receives a request.
func (cl *WssAgentTransport) HandleAgentMessage(baseMsg wssserver_old.BaseMessage, raw []byte) {
	var req transports.RequestMessage
	var isRequest = true
	var err error

	if cl.AppRequestHandler == nil {
		slog.Error("HandleAgentMessage: no request handler set",
			"clientID", cl.GetClientID(),
			"msgType", baseMsg.MessageType,
		)
		return
	}
	isRequest, req = cl.wssToRequest(baseMsg, raw)
	if !isRequest {
		slog.Info("HandleAgentMessage: not a request. Ignored",
			slog.String("clientID", cl.GetClientID()),
			slog.String("msgType", baseMsg.MessageType),
			slog.String("correlationID", baseMsg.CorrelationID),
		)
		return
	}
	slog.Info("HandleAgentMessage",
		slog.String("clientID", cl.GetClientID()),
		slog.String("msgType", baseMsg.MessageType),
		slog.String("correlationID", baseMsg.CorrelationID),
	)

	resp := cl.AppRequestHandler(req)

	// send the response to the caller
	err = cl.SendResponse(resp)
	if err != nil {
		slog.Error("handleAgentRequest: failed", "err", err.Error())
	}
}

// Init Initializes the HTTP/websocket consumer client transport
// For internal use in the construction phase.
//
//	fullURL full path of the sse endpoint
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func (cl *WssAgentTransport) Init(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) {

	// forms are not used in agents
	cl.WssConsumerClient.Init(
		fullURL, clientID, clientCert, caCert, nil, timeout)
	cl.agentRequestHandler = cl.HandleAgentMessage
}

// SendNotification sends an operation without expecting a respones
func (cl *WssAgentTransport) SendNotification(notif transports.NotificationMessage) error {

	// convert the operation into a websocket message
	wssMsg, err := wssserver_old.OpToMessage(
		notif.Operation, notif.ThingID, notif.Name, nil, notif.Data,
		"", cl.GetClientID())
	if err != nil {
		slog.Error("SendNotification: failed", "err", err.Error())
		return err
	}
	err = cl._send(wssMsg)
	return err
}

// SetRequestHandler set the application handler for incoming requests
func (cl *WssAgentTransport) SetRequestHandler(cb transports.RequestHandler) {
	cl.AppRequestHandler = cb
}

// SendResponse Agent sends a response to a request.
func (cl *WssAgentTransport) SendResponse(resp *transports.ResponseMessage) (err error) {

	var wssMsg any

	slog.Debug("SendResponse",
		slog.String("agentID", cl.BaseClientID),
		slog.String("thingID", resp.ThingID),
		slog.String("name", resp.Name),
		slog.String("correlationID", resp.CorrelationID))

	// convert the operation into a websocket message
	if resp.Error == "" {
		wssMsg, err = wssserver_old.OpToMessage(
			resp.Operation, resp.ThingID, resp.Name, nil, resp.Output,
			"", cl.GetClientID())
	} else {
		wssMsg = wssserver_old.ErrorMessage{
			ThingID:       resp.ThingID,
			Name:          resp.Name,
			MessageType:   wssserver_old.MsgTypeError,
			Title:         resp.Error,
			Detail:        fmt.Sprintf("%v", resp.Output),
			Status:        transports.StatusFailed,
			CorrelationID: resp.CorrelationID,
			Timestamp:     resp.Updated,
		}
	}
	err = cl._send(wssMsg)
	if err != nil {
		slog.Error("SendNotification: failed", "err", err.Error())
		return err
	}
	return err
}

// NewWssAgentClient creates a new instance of the websocket hub client.
func NewWssAgentClient(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) *WssAgentTransport {
	cl := WssAgentTransport{}
	cl.Init(fullURL, clientID, clientCert, caCert, timeout)

	return &cl
}
