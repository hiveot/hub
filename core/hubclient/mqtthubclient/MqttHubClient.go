package mqtthubclient

import (
	"errors"
	"fmt"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
	"strings"
	"time"
)

// higher level Hub event and action functions

// MqttHubSubscription  subscription helper
// This implements ISubscription
type MqttHubSubscription struct {
	topic   string
	handler func(topic string, payload []byte)
	client  *paho.Client
}

// MqttHubClient manages the hub server connection with hub event and action messaging
// This implements the IHubClient interface.
// This implementation is based on the Mqtt messaging system.
type MqttHubClient struct {
	clientID string
	hostName string
	port     int
	cm       *autopaho.ConnectionManager
	router   paho.Router
}

// ClientID the client is authenticated as to the server
func (hc *MqttHubClient) ClientID() string {
	return hc.clientID
}

// ParseResponse helper message to parse response and detect the error response message
func (hc *MqttHubClient) ParseResponse(data []byte, err error, resp interface{}) error {
	if err != nil {
		return err
	}
	if resp != nil {
		err = ser.Unmarshal(data, resp)
	} else if string(data) == "+ACK" {
		err = nil
	} else if len(data) > 0 {
		err = errors.New("unexpected response")
	}
	// if an error is detect see if it is an error response
	// An error response message has the format: {"error":"message"}
	// TODO: find a more idiomatic way to detect an error
	prefix := "{\"error\":"
	if err != nil || strings.HasPrefix(string(data), prefix) {
		errResp := hubclient.ErrorMessage{}
		err2 := ser.Unmarshal(data, &errResp)
		if err2 == nil && errResp.Error != "" {
			err = errors.New(errResp.Error)
		}
	}
	return err
}

// PubThingAction sends an action request to the hub and receives a response
// Returns the response or an error if the request fails or timed out
func (hc *MqttHubClient) PubThingAction(bindingID string, thingID string, actionID string, payload []byte) ([]byte, error) {
	topic := MakeThingsTopic(bindingID, thingID, actionID, hc.clientID)
	slog.Info("PubThingAction", "topic", topic)
	resp, err := hc.Request(topic, payload)
	if resp == nil {
		return nil, err
	}
	return resp, err
}

// PubServiceAction sends an action request to a Hub Service on the svc prefix
// Returns the response or an error if the request fails or timed out
func (hc *MqttHubClient) PubServiceAction(serviceID string, capability string, actionID string, payload []byte) ([]byte, error) {
	topic := MakeServiceActionTopic(serviceID, capability, actionID, hc.clientID)
	slog.Info("PubServiceAction", "topic", topic)
	resp, err := hc.Request(topic, payload)
	if resp == nil {
		return nil, err
	}
	return resp, err
}

// PubEvent sends the event value to the hub
func (hc *MqttHubClient) PubEvent(thingID string, eventID string, payload []byte) error {
	topic := MakeThingsTopic(hc.clientID, thingID, vocab.MessageTypeEvent, eventID)
	slog.Info("PubEvent", "topic", topic)
	err := hc.Pub(topic, payload)
	return err
}

// PubTD sends the TD document to the hub
func (hc *MqttHubClient) PubTD(td *thing.TD) error {
	payload, _ := ser.Marshal(td)
	topic := MakeThingsTopic(hc.clientID, td.ID, vocab.MessageTypeEvent, vocab.EventNameTD)
	slog.Info("PubTD", "topic", topic)
	err := hc.Pub(topic, payload)
	return err
}

// subscribe to topics after establishing connection
// The application can already subscribe to topics before the connection is established. If connection is lost then
// this will re-subscribe to those topics as PahoMqtt drops the subscriptions after disconnect.
//func (hc *MqttHubClient) resubscribe() {
//	//
//	slog.Info("mqtt.resubscribe to topics", "n", len(mqttClient.subscriptions))
//	for _, subscription := range mqttClient.subscriptions {
//		// clear existing subscription
//		hc.pahoClient.Unsubscribe(subscription.topic)
//
//		// create a new variable to hold the subscription in the closure
//		newSubscr := subscription
//		token := hc.pahoClient.Subscribe(newSubscr.topic, newSubscr.qos, newSubscr.onMessage)
//		//token := mqttClient.pahoClient.Subscribe(newSubscr.topic, newSubscr.qos, func (c pahomqtt.Client, msg pahomqtt.Message) {
//		//mqttClient.log.Infof("mqtt.resubscribe.onMessage: topic %s, subscription %s", msg.Topic(), newSubscr.topic)
//		//newSubscr.onMessage(c, msg)
//		//})
//		newSubscr.token = token
//	}
//}

// Refresh an authentication token.
//
//	clientID is the userID, deviceID or serviceID whose token to refresh.
//	oldToken must be a valid token obtained at login or refresh
//
// This returns a short lived auth token that can be used to authenticate with the hub
// This fails if the token has expired or does not belong to the clientID
//func (hc *MqttHubClient) Refresh(clientID string, oldToken string) (newToken string, err error) {
//	req := &authn.RefreshReq{
//		UserID: clientID,
//		OldToken: oldToken,
//	}
//	msg, _ := ser.Marshal(req)
//	topic := MakeThingsTopic(hc.clientID, td.ID, vocab.MessageTypeEvent, vocab.EventNameTD)
//	slog.Info("PubTD", "topic", topic)
//	err := hc.Publish(topic, payload)
//	resp := &authn.RefreshResp{}
//	err = hubclient.ParseResponse(data, err, resp)
//	if err == nil {
//		authToken = resp.JwtToken
//	}
//	return err
//}

// SubActions subscribes to actions on the given topic
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
func (hc *MqttHubClient) SubActions(
	thingID string, cb func(msg *hubclient.ActionRequest) error,
) (hubclient.ISubscription, error) {

	topic := MakeThingActionTopic(hc.clientID, thingID, "", "")
	sub, err := hc.SubRequest(topic, func(pahoMsg *PahoMsg) error {
		timeStamp := time.Now()
		deviceID, thID, name, clientID, err := SplitActionTopic(topic)
		if err != nil {
			slog.Error("unable to handle topic", "err", err, "topic", topic)
			return err
		}
		actionMsg := &hubclient.ActionRequest{
			ClientID:  clientID,
			ActionID:  name,
			DeviceID:  deviceID,
			ThingID:   thID,
			Timestamp: timeStamp.Unix(),
			Payload:   pahoMsg.Payload,
			SendReply: func(payload []byte) {
				pahoMsg.SendReply(payload)
			},
			SendAck: func() {
				pahoMsg.SendReply(nil)
			},
		}
		err = cb(actionMsg)
		return err
	})
	return sub, err
}

func (hc *MqttHubClient) SubStream(
	name string, receiveLatest bool, cb func(msg *hubclient.EventMessage)) (hubclient.ISubscription, error) {
	return nil, fmt.Errorf("not implemented")
}

// SubThingActions subscribes to actions for this device or service on the things prefix
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
func (hc *MqttHubClient) SubThingActions(
	thingID string, cb func(msg *hubclient.ActionRequest) error,
) (hubclient.ISubscription, error) {

	topic := MakeThingActionTopic(hc.clientID, thingID, "", "")
	return hc.SubActions(topic, cb)
}

// SubServiceActions subscribes to action requests of a service capability
//
//	capability is the name of the capability (thingID) to handle
func (hc *MqttHubClient) SubServiceActions(
	capability string, cb func(msg *hubclient.ActionRequest) error,
) (hubclient.ISubscription, error) {

	topic := MakeServiceActionTopic(hc.clientID, capability, "", "")
	return hc.SubActions(topic, cb)
}

// SubStream subscribes to events received by the event stream.
//
// This creates an ephemeral pull consumer.
// ReceiveLatest is handy to be up to date on all event instead of quering them separately. Only use this if
// you're going to retrieve them anyways.
//
//	 name of the event stream. "" for default
//		receiveLatest to immediately receive the latest event for each event instance
//func (hc *MqttHubClient) SubStream(name string, receiveLatest bool, cb func(msg *hubclient.EventMessage)) (hubclient.ISubscription, error) {
//
//}

// NewMqttHubClient instantiates a client for connecting to the Hub using MQTT
// Intended for use by the 'ConnectXyz' methods that define the authentication options
func NewMqttHubClient() *MqttHubClient {

	hc := &MqttHubClient{}
	return hc
}
