package mqttclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot/td"
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

// PubEvent publishes a message and returns
func (cl *MqttAgentClient) PubEvent(topic string, payload []byte) (err error) {
	slog.Debug("PubEvent", "topic", topic)
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

// SendResponse sends the action response message
func (cl *MqttAgentClient) SendResponse(resp transports.ResponseMessage) error {
	// FIXME: in paho, the response needs a reply-to address from the request
	// option 1: include this in the requestID
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
	operation string, thingID, name string, input interface{}) error {

	// FIXME: implement message envelope
	err := errors.New("not implemented")
	return err
}

// SetRequestHandler set the application handler for incoming requests
func (cl *MqttAgentClient) SetRequestHandler(cb transports.RequestHandler) {
	cl.appRequestHandler = cb
}

// Initialize the client
func (cl *MqttAgentClient) Init(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm func(op string) td.Form,
	timeout time.Duration) {

	cl.MqttConsumerClient.Init(
		fullURL, clientID, clientCert, caCert, getForm, timeout)

	//cl.BaseHandleMessage = cl.HandleAgentMessage
}

// NewMqttAgentTransport creates a new instance of the mqtt binding client
//
//	fullURL of broker to connect to, including the schema
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	getForm is the handler that provides a form for the given operation
//	timeout for waiting for response. 0 to use the default.
func NewMqttAgentTransport(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm func(op string) td.Form,
	timeout time.Duration) *MqttAgentClient {

	cl := MqttAgentClient{}
	cl.Init(fullURL, clientID, clientCert, caCert, getForm, timeout)

	return &cl
}
