package natshubclient

import (
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/ser"
	"github.com/nats-io/nats.go"
	"golang.org/x/exp/slog"
	"time"
)

// Higher level Hub event and action functions

// NatsHubClient manages the hub server connection with hub event and action messaging
// This implements the IHubClient interface.
// This implementation is based on the NATS/Jetstream messaging system.
type NatsHubClient struct {
	clientID string
	nc       *nats.Conn
	js       nats.JetStreamContext
	timeout  time.Duration
}

// ClientID the client is authenticated as to the server
func (hc *NatsHubClient) ClientID() string {
	return hc.clientID
}

// PubThingAction sends an action request to the hub and receives a response
// Returns the response or an error if the request fails or timed out
func (hc *NatsHubClient) PubThingAction(bindingID string, thingID string, actionID string, payload []byte) ([]byte, error) {
	subject := MakeThingActionSubject(bindingID, thingID, actionID, hc.clientID)
	slog.Info("PubThingAction", "subject", subject)
	resp, err := hc.nc.Request(subject, payload, hc.timeout)
	if resp == nil {
		return nil, err
	}
	return resp.Data, err
}

// PubServiceAction sends an action request to a Hub Service on the svc prefix
// Returns the response or an error if the request fails or timed out
func (hc *NatsHubClient) PubServiceAction(serviceID string, capability string, actionID string, payload []byte) ([]byte, error) {
	subject := MakeServiceActionSubject(serviceID, capability, actionID, hc.clientID)
	slog.Info("PubServiceAction", "subject", subject)
	resp, err := hc.nc.Request(subject, payload, hc.timeout)
	if resp == nil {
		return nil, err
	}
	return resp.Data, err
}

// PubEvent sends the event value to the hub
func (hc *NatsHubClient) PubEvent(thingID string, eventID string, payload []byte) error {
	subject := MakeThingsSubject(hc.clientID, thingID, vocab.MessageTypeEvent, eventID)
	slog.Info("PubEvent", "subject", subject)
	err := hc.nc.Publish(subject, payload)
	return err
}

// PubTD sends the TD document to the hub
func (hc *NatsHubClient) PubTD(td *thing.TD) error {
	payload, _ := ser.Marshal(td)
	subject := MakeThingsSubject(hc.clientID, td.ID, vocab.MessageTypeEvent, vocab.EventNameTD)
	slog.Info("PubTD", "subject", subject)
	err := hc.nc.Publish(subject, payload)
	return err
}

// Refresh an authentication token.
//
//	clientID is the userID, deviceID or serviceID whose token to refresh.
//	oldToken must be a valid token obtained at login or refresh
//
// This returns a short lived auth token that can be used to authenticate with the hub
// This fails if the token has expired or does not belong to the clientID
//func (hc *NatsHubClient) Refresh(clientID string, oldToken string) (newToken string, err error) {
//	req := &authn.RefreshReq{
//		UserID: clientID,
//		OldToken: oldToken,
//	}
//	msg, _ := ser.Marshal(req)
//	subject := MakeThingsSubject(hc.clientID, td.ID, vocab.MessageTypeEvent, vocab.EventNameTD)
//	slog.Info("PubTD", "subject", subject)
//	err := hc.Publish(subject, payload)
//	resp := &authn.RefreshResp{}
//	err = hubclient.ParseResponse(data, err, resp)
//	if err == nil {
//		authToken = resp.JwtToken
//	}
//	return err
//}

// SubActions subscribes to actions on the given subject
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
func (hc *NatsHubClient) SubActions(
	subject string, cb func(msg *hubclient.ActionRequest) error) (
	hubclient.ISubscription, error) {

	sub, err := hc.Subscribe(subject, func(natsMsg *nats.Msg) {
		md, _ := natsMsg.Metadata()
		timeStamp := time.Now()
		if md != nil {
			timeStamp = md.Timestamp

		}
		payload := natsMsg.Data
		deviceID, thID, name, clientID, err := SplitActionSubject(natsMsg.Subject)
		if err != nil {
			slog.Error("unable to handle subject", "err", err, "subject", natsMsg.Subject)
			return
		}
		actionMsg := &hubclient.ActionRequest{
			//SenderID: natsMsg.Header.
			ClientID:  clientID,
			ActionID:  name,
			DeviceID:  deviceID,
			ThingID:   thID,
			Timestamp: timeStamp.Unix(),
			Payload:   payload,
			SendReply: func(payload []byte, err error) error {
				if err != nil {
					errMsg := hubclient.ErrorMessage{Error: err.Error()}
					payload, _ = ser.Marshal(errMsg)
				}
				return natsMsg.Respond(payload)
			},
			SendAck: func() error {
				return natsMsg.Ack()
			},
		}
		err = cb(actionMsg)
		if err != nil {
			errMsg := hubclient.ErrorMessage{Error: err.Error()}
			errPayload, _ := ser.Marshal(errMsg)
			_ = natsMsg.Respond(errPayload)
		}
	})
	return sub, err
}

// SubThingActions subscribes to action requests of a device's Thing.
// Intended for use by device implementors to receive requests for its things.
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
func (hc *NatsHubClient) SubThingActions(
	thingID string, cb func(msg *hubclient.ActionRequest) error) (hubclient.ISubscription, error) {

	subject := MakeThingActionSubject(hc.clientID, thingID, "", "")
	return hc.SubActions(subject, cb)
}

// SubServiceActions subscribes to service RPC requests.
// Intended for use by services to receive requests.
//
//	capability is the name of the capability (thingID) to handle
func (hc *NatsHubClient) SubServiceActions(
	capability string, cb func(msg *hubclient.ActionRequest) error) (hubclient.ISubscription, error) {

	subject := MakeServiceActionSubject(hc.clientID, capability, "", "")
	return hc.SubActions(subject, cb)
}

// SubEvents subscribe to event
//func (hc *NatsHubClient) SubEvents(thingID string, cb func(msg *hubclient.EventMessage)) (hubclient.ISubscription, error) {
//
//	subject := MakeThingsSubject(hc.clientID, thingID, "event", "")
//
//	sub, err := hc.Subscribe(subject, func(natsMsg *nats.Msg) {
//		md, _ := natsMsg.Metadata()
//		timeStamp := time.Now()
//		if md != nil {
//			timeStamp = md.Timestamp
//
//		}
//		payload := natsMsg.Data
//		bindingID, thID, _, name, err := SplitSubject(natsMsg.Subject)
//		if err != nil {
//			slog.Error("unable to handle subject", "err", err, "subject", natsMsg.Subject)
//			return
//		}
//		eventMsg := &hubclient.EventMessage{
//			BindingID: bindingID,
//			ThingID:   thID,
//			EventID:   name,
//			Timestamp: timeStamp.Unix(),
//			Payload:   payload,
//		}
//		cb(eventMsg)
//	})
//	return sub, err
//}

// NewHubClient instantiates a client for connecting to the Hub using NATS/Jetstream
func NewHubClient(clientID string, nc *nats.Conn) (hc *NatsHubClient, err error) {

	hc = &NatsHubClient{
		clientID: clientID,
		nc:       nc,
		timeout:  time.Duration(DefaultTimeoutSec) * time.Second,
	}
	hc.js, err = hc.nc.JetStream()
	hc.timeout = time.Duration(10) * time.Second // for testing
	return hc, err
}
