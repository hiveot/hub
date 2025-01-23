package mqttclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"time"
)

// MqttAgentClient provides WoT protocol binding for the MQTT protocol
// This implements the IAgentTransport interface.
type MqttAgentClient struct {
	MqttConsumerClient

	// the application's request handler set with SetRequestHandler
	appRequestHandler func(msg transports.RequestMessage) (response transports.ResponseMessage)
}

// handleMessage handles incoming MQTT messages from the agent
// FIXME: rework for hiveot
//func (cl *MqttAgentClient) HandleAgentMessage(msg *transports.ThingMessage) {
//	slog.Debug("handleMessage", slog.String("op", msg.Operation))
//	cl.HandleConsumerMessage(msg)
//}

// PubEvent helper for agents to publish an event
// This is short for SendNotification( ... wot.OpEvent ...)
func (cl *MqttAgentClient) PubEvent(thingID string, name string, value any) error {
	notif := transports.NewNotificationResponse(wot.HTOpEvent, thingID, name, value)
	return cl.SendNotification(notif)
}

// PubProperty helper for agents to publish a property value update
// This is short for SendNotification( ... wot.OpProperty ...)
func (cl *MqttAgentClient) PubProperty(thingID string, name string, value any) error {

	notif := transports.NewNotificationResponse(wot.HTOpUpdateProperty, thingID, name, value)
	return cl.SendNotification(notif)
}

// PubProperties helper for agents to publish a map of property values
func (cl *MqttAgentClient) PubProperties(thingID string, propMap map[string]any) error {

	notif := transports.NewNotificationResponse(wot.HTOpUpdateMultipleProperties, thingID, "", propMap)
	err := cl.SendNotification(notif)
	return err
}

// PubTD helper for agents to publish a TD update
// This is short for SendNotification( ... wot.HTOpTD ...)
func (cl *MqttAgentClient) PubTD(td *td.TD) error {
	tdJson, _ := jsoniter.Marshal(td)
	notif := transports.NewNotificationResponse(wot.HTOpUpdateTD, td.ID, "", tdJson)
	return cl.SendNotification(notif)
}

//// PubEvent publishes a message and returns
//func (cl *MqttAgentClient) PubEvent(topic string, payload []byte) (err error) {
//	slog.Debug("PubEvent", "topic", topic)
//	ctx, cancelFn := context.WithTimeout(context.Background(), cl.BaseTimeout)
//	defer cancelFn()
//	pubMsg := &paho.Publish{
//		QoS:     0, //withQos,
//		Retain:  false,
//		Topic:   topic,
//		Payload: payload,
//	}
//	cl.BaseMux.RLock()
//	pcl := cl.pahoClient
//	cl.BaseMux.RUnlock()
//	if pcl != nil {
//		_, err = pcl.Publish(ctx, pubMsg)
//	} else {
//		err = errors.New("no connection with the hub")
//	}
//	return err
//}

// SendResponse sends the action response message
func (cl *MqttAgentClient) SendResponse(resp transports.ResponseMessage) error {
	// FIXME: in paho, the response needs a reply-to address from the request
	// option 1: include this in the correlationID
	// option 2: use context to pass it
	//err = cl._send(resp)
	//return err
	return fmt.Errorf("Not implemented")
}

// sendReply sends a reply on the response topic of the request
// This uses the same QoS as the request, without retain.
//
//	req is the request to reply to
//	optionally include a payload in the reply
//	optionally include an error message in the reply
func (cl *MqttAgentClient) sendReply(req *paho.Publish, payload []byte, errResp error) (err error) {

	slog.Debug("sendReply",
		slog.String("topic", req.Topic),
		slog.String("responseTopic", req.Properties.ResponseTopic))

	responseTopic := req.Properties.ResponseTopic
	if responseTopic == "" {
		err2 := fmt.Errorf("sendReply. No response topic. Not sending a reply")
		slog.Error(err2.Error())
	}
	replyMsg := &paho.Publish{
		QoS:    req.QoS,
		Retain: false,
		Topic:  responseTopic,
		Properties: &paho.PublishProperties{
			CorrelationData: req.Properties.CorrelationData,
			User:            req.Properties.User,
			PayloadFormat:   req.Properties.PayloadFormat,
			ContentType:     req.Properties.ContentType,
		},
		Payload: payload,
	}
	if errResp != nil {
		replyMsg.Properties.ContentType = "error" // payload is an error message
		replyMsg.Properties.User.Add("error", errResp.Error())
		// for testing, somehow properties.user is not transferred
		replyMsg.Payload = []byte(errResp.Error())
	}
	cl.BaseMux.RLock()
	pcl := cl.pahoClient
	cl.BaseMux.RUnlock()
	if pcl == nil {
		err = errors.New("connection lost")
	} else {
		ctx, cancelFn := context.WithTimeout(context.Background(), time.Second)
		defer cancelFn()
		_, err = pcl.Publish(ctx, replyMsg)

		if err != nil {
			slog.Warn("sendReply. Error publishing response",
				slog.String("err", err.Error()))
		}
	}
	return err
}

// SendNotification sends the operation as a notification and returns immediately.
func (cl *MqttAgentClient) SendNotification(
	notif transports.NotificationMessage) (err error) {

	topic := "hiveot/notification"
	payload, _ := jsoniter.Marshal(notif)
	cl._send(topic, payload, "")
	slog.Debug("SendNotification", "topic", topic)
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.BaseTimeout)
	defer cancelFn()
	pubMsg := &paho.Publish{
		QoS:     0, //withQos,
		Retain:  false,
		Topic:   topic,
		Payload: payload,
	}
	cl.BaseMux.RLock()
	pcl := cl.pahoClient
	cl.BaseMux.RUnlock()
	if pcl != nil {
		_, err = pcl.Publish(ctx, pubMsg)
	} else {
		err = errors.New("no connection with the hub")
	}
	return err
}

// SetRequestHandler set the application handler for incoming requests
func (cl *MqttAgentClient) SetRequestHandler(cb transports.RequestHandler) {
	cl.appRequestHandler = cb
}

// Initialize the client
func (cl *MqttAgentClient) Init(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) {

	cl.MqttConsumerClient.Init(
		fullURL, clientID, clientCert, caCert, nil, timeout)

	//cl.BaseHandleMessage = cl.HandleAgentMessage
}

// NewMqttAgentClient creates a new instance of the mqtt binding client
//
//	fullURL of broker to connect to, including the schema
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func NewMqttAgentClient(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) *MqttAgentClient {

	cl := MqttAgentClient{}
	cl.Init(fullURL, clientID, clientCert, caCert, timeout)

	return &cl
}
