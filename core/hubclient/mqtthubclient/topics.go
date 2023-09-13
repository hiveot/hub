package mqtthubclient

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"strings"
)

// MakeThingsTopic creates a nats topic optionally with nats wildcards
// This uses the hiveot nats topic format: things.{pubID}.{thingID}.{type}.{name}
//
//	pubID is the publisher of the topic. Use "" for wildcard
//	thingID is the ID of the thing managed by the publisher. Use "" for wildcard
//	stype is the topic type: "event" or "action".
//	name is the event or action name. Use "" for wildcard.
func MakeThingsTopic(pubID, thingID, stype, name string) string {
	if pubID == "" {
		pubID = "+"
	}
	if thingID == "" {
		thingID = "+" // nats uses *
	}
	if stype == "" {
		stype = vocab.MessageTypeEvent
	}
	if name == "" {
		name = "#" // anything after is fine
	}
	subj := fmt.Sprintf("things/%s/%s/%s/%s",
		pubID, thingID, stype, name)
	return subj
}

// MakeServiceTopic creates a nats topic optionally with nats wildcards
// This uses the hiveot nats topic format: svc.{serviceID}.{capName}.{type}.{name}
//
//	serviceID is the publisher ID of the service
//	capID is the capabilities ID provided by the service. Use "" for wildcard
//	stype is the topic type: "event" or "action".
//	name is the event or action name. Use "" for wildcard.
func MakeServiceTopic(serviceID, capID, stype, name string) string {
	if serviceID == "" {
		serviceID = "+"
	}
	if capID == "" {
		capID = "+"
	}
	if stype == "" {
		stype = vocab.MessageTypeEvent
	}
	if name == "" {
		name = "#" // anything after is fine
	}
	topic := fmt.Sprintf("svc/%s/%s/%s/%s",
		serviceID, capID, stype, name)
	return topic
}

// MakeThingActionTopic creates a nats topic for submitting actions
// This uses the hiveot nats topic format: things.{bindingID}.{thingID}.action.{name}.{clientID}
//
//	deviceID is the publisher of the topic. Use "" for wildcard
//	thingID is the ID of the thing managed by the publisher. Use "" for wildcard
//	name is the event or action name. Use "" for wildcard.
//	senderID is the loginID of the user submitting the action
func MakeThingActionTopic(deviceID, thingID, name string, senderID string) string {
	if deviceID == "" {
		deviceID = "+"
	}
	if thingID == "" {
		thingID = "+"
	}
	if name == "" {
		name = "+"
	}
	if senderID == "" {
		senderID = "+"
	}
	topic := fmt.Sprintf("things/%s/%s/%s/%s/%s",
		deviceID, thingID, vocab.MessageTypeAction, name, senderID)
	return topic
}

// MakeServiceActionTopic creates a nats topic for submitting service requests
// This uses the hiveot nats topic format: things.{bindingID}.{thingID}.action.{name}.{clientID}
//
//	serviceID is the publisher of the topic. Use "" for wildcard
//	thingID is the ID of the thing managed by the publisher. Use "" for wildcard
//	name is the event or action name. Use "" for wildcard.
//	senderID is the loginID of the user submitting the action
func MakeServiceActionTopic(serviceID, thingID, name string, senderID string) string {
	if serviceID == "" {
		serviceID = "+"
	}
	if thingID == "" {
		thingID = "+"
	}
	if name == "" {
		name = "+"
	}
	if senderID == "" {
		senderID = "+"
	}
	topic := fmt.Sprintf("svc/%s/%s/%s/%s/%s",
		serviceID, thingID, vocab.MessageTypeAction, name, senderID)
	return topic
}

// SplitActionTopic separates a topic into its components
//
// topic is a hiveot nats topic. eg: things.bindingID.thingID.stype.name.clientID
//
//	bindingID is the device or service that handles the topic.
//	thingID is the thing of the topic, or capability for services.
//	stype is the topic type, eg event or action.
//	name is the action name
//	clientID is the client that publishes the action. This identifies the publisher.
func SplitActionTopic(topic string) (bindingID, thingID, name string, clientID string, err error) {
	parts := strings.Split(topic, "/")
	if len(parts) < 6 {
		err = errors.New("incomplete topic")
		return
	}
	stype := parts[3]
	if stype != "action" {
		err = fmt.Errorf("topic %s is not an action", topic)
		return
	}
	bindingID = parts[1]
	thingID = parts[2]
	name = parts[4]
	clientID = parts[5]
	return
}

// SplitTopic separates a topic into its components
//
// topic is a hiveot nats topic. eg: things.publisherID.thingID.type.name
//
//		prefix of things or services
//		deviceID is the device or service that handles the topic.
//		thingID is the thing of the topic, or capability for services.
//		stype is the topic type, eg event or action.
//		name is the event or action name
//	 clientID is the ID in the address suffix. This is used in actions, not in events
func SplitTopic(topic string) (prefix, deviceID, thingID, stype, name string, clientID string, err error) {
	parts := strings.Split(topic, "/")
	if len(parts) < 5 {
		err = errors.New("incomplete topic")
		return
	}
	prefix = parts[0]
	deviceID = parts[1]
	thingID = parts[2]
	stype = parts[3]
	name = parts[4]
	if len(parts) > 5 {
		clientID = parts[5]
	}
	return
}
