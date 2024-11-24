package mqttbinding

import (
	"context"
	"errors"
	"fmt"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/wot/transports"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"strings"
	"time"
)

// handleMessage handles incoming MQTT messages from the agent
// FIXME: rework for hiveot
func (cl *MqttBindingClient) handleMessage(m *paho.Publish) {
	slog.Debug("handleMessage", slog.String("topic", m.Topic))
	// run this in the background to allow for reentrancy
	go func() {
		// handle reply message
		if strings.HasPrefix(m.Topic, INBOX_PREFIX) && m.Properties.CorrelationData != nil {
			// Pass replies to their waiting channel
			cID := string(m.Properties.CorrelationData)
			cl.mux.RLock()
			rChan, _ := cl.correlData[cID]
			cl.mux.RUnlock()
			if rChan == nil {
				slog.Warn("Received reply without matching correlation ID", "corrID", cID)
			} else {
				cl.mux.Lock()
				delete(cl.correlData, cID)
				cl.mux.Unlock()

				rChan <- m
			}
			return
		}

		// handle request message
		replyTo := m.Properties.ResponseTopic
		if replyTo != "" && m.Properties.CorrelationData != nil {
			var reply []byte
			var err error
			var donotreply bool
			// get a reply from the single request handler
			cl.mux.RLock()
			reqHandler := cl.requestHandler
			cl.mux.RUnlock()

			if reqHandler != nil {
				//reply, err, donotreply = reqHandler(m.Topic, m.Payload)
				if err != nil {
					slog.Warn("SubRequest: handle request failed.",
						slog.String("err", err.Error()),
						slog.String("topic", m.Topic))
				}
			} else {
				slog.Error("Received request message but no request handler is set.",
					slog.String("clientID", cl.clientID),
					slog.String("topic", m.Topic),
					slog.String("replyTo", replyTo))
				err = errors.New("Cannot handle request. No handler is set")
			}
			if !donotreply {
				err = cl.sendReply(m, reply, err)

				if err != nil {
					slog.Error("SubRequest. Sending reply failed", "err", err)
				}
			}
		} else {
			// this is en event message
			cl.mux.RLock()
			evHandler := cl.messageHandler
			cl.mux.RUnlock()
			if evHandler != nil {
				//evHandler(m.Topic, m.Payload)
				op := ""
				thingID := ""
				name := ""
				data := m.Payload
				senderID := ""
				tm := transports.NewThingMessage(op, thingID, name, data, senderID)
				evHandler(tm)
			}
		}
	}()
}

// ParseResponse helper message to parse response and check for errors
func (cl *MqttBindingClient) ParseResponse(data []byte, resp interface{}) error {
	var err error
	if data == nil || len(data) == 0 {
		if resp != nil {
			err = fmt.Errorf("ParseResponse: client '%s', expected a response but none received", cl.clientID)
		} else {
			err = nil // all good
		}
	} else {
		if resp == nil {
			err = fmt.Errorf("ParseResponse: client '%s', received response but none was expected. data=%s",
				cl.clientID, data)
		} else {
			err = jsoniter.Unmarshal(data, resp)
		}
	}
	return err
}

// PubEvent publishes a message and returns
func (cl *MqttBindingClient) PubEvent(topic string, payload []byte) (err error) {
	slog.Debug("PubEvent", "topic", topic)
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	defer cancelFn()
	pubMsg := &paho.Publish{
		QoS:     0, //withQos,
		Retain:  false,
		Topic:   topic,
		Payload: payload,
	}
	cl.mux.RLock()
	pcl := cl.pahoClient
	cl.mux.RUnlock()
	if pcl != nil {
		_, err = pcl.Publish(ctx, pubMsg)
	} else {
		err = errors.New("no connection with the hub")
	}
	return err
}

// PubRequest publishes a request message and waits for an answer or until timeout
// In order to receive replies, an inbox subscription is added on the first request.
func (cl *MqttBindingClient) PubRequest(topic string, payload []byte) (resp []byte, err error) {
	slog.Debug("PubRequest", "topic", topic)

	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	defer cancelFn()

	// FIXME! a deadlock can occur here
	cl.mux.RLock()
	pcl := cl.pahoClient
	inboxTopic := cl.inboxTopic
	connectID := cl.connectID
	cl.mux.RUnlock()

	if pcl == nil {
		return nil, fmt.Errorf("connection lost")
	}
	if inboxTopic == "" {
		inboxTopic = fmt.Sprintf(InboxTopicFormat, connectID)
		if connectID == "" {
			err = fmt.Errorf("can't publish request as connectID is not set. This is unexpected.")
			slog.Error(err.Error())
			return nil, err
		}
		cl.mux.Lock()
		cl.inboxTopic = inboxTopic
		cl.mux.Unlock()
		err = cl.Subscribe(inboxTopic)
		if err != nil {
			slog.Error("Failed inbox subscription",
				"err", err, "inboxTopic", inboxTopic)
			return nil, err
		}
	}
	// from paho rpc.go:
	cid := fmt.Sprintf("%d", time.Now().UnixNano())
	rChan := make(chan *paho.Publish)
	cl.mux.Lock()
	cl.correlData[cid] = rChan
	cl.mux.Unlock()

	pubMsg := &paho.Publish{
		QoS:     withQos,
		Retain:  false,
		Topic:   topic,
		Payload: payload,
		Properties: &paho.PublishProperties{
			CorrelationData: []byte(cid),
			ResponseTopic:   inboxTopic,
			ContentType:     "json",
			User: paho.UserProperties{{
				Key:   "test",
				Value: "test",
			}},
		},
	}
	_, err = pcl.Publish(ctx, pubMsg)
	if err != nil {
		return nil, err
	}

	// wait for response
	var respMsg *paho.Publish
	select {
	case <-ctx.Done():
		err = fmt.Errorf("timeout waiting for response")
		break
	case respMsg = <-rChan:
		break
	}
	if err != nil {
		return nil, err
	}

	// test alternative to handling errors since User properties aren't
	// passed through for some reason.
	if respMsg.Properties.ContentType == "error" {
		err = errors.New(string(respMsg.Payload))
		return nil, err
	}

	slog.Debug("PubRequest end:",
		slog.String("topic", topic),
		slog.String("ContentType (if any)", respMsg.Properties.ContentType),
	)
	return respMsg.Payload, err
}

// sendReply sends a reply on the response topic of the request
// This uses the same QoS as the request, without retain.
//
//	req is the request to reply to
//	optionally include a payload in the reply
//	optionally include an error message in the reply
func (cl *MqttBindingClient) sendReply(req *paho.Publish, payload []byte, errResp error) (err error) {

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
	cl.mux.RLock()
	pcl := cl.pahoClient
	cl.mux.RUnlock()
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

// Subscribe subscribes to a topic.
// Incoming messages are passed to the event or request handler, depending on whether
// a reply-to address and correlation-ID is set.
func (cl *MqttBindingClient) Subscribe(topic string) error {
	slog.Debug("Subscribe", "topic", topic)
	err := cl.sub(topic)
	if err != nil {
		return err
	}
	cl.mux.Lock()
	cl.subscriptions[topic] = true
	cl.mux.Unlock()
	return err
}
func (cl *MqttBindingClient) Unsubscribe(topic string) {
	packet := &paho.Unsubscribe{
		Topics: []string{topic},
	}
	cl.mux.RLock()
	pcl := cl.pahoClient
	cl.mux.RUnlock()

	ack, err := pcl.Unsubscribe(context.Background(), packet)
	_ = ack
	if err != nil {
		slog.Error("Unable to unsubscribe from topic", "topic", topic)
		return
	}
	cl.mux.Lock()
	delete(cl.subscriptions, topic)
	cl.mux.Unlock()
}
