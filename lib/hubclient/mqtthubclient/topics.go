package mqtthubclient

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"strings"
)

// MakeTopic creates a mqtt message  topic optionally with wildcards
// This uses the hiveot topic format: {msgType}/{deviceID}/{thingID}/{name}[/{clientID}]
//
//	msgType is the message type: "event", "action", "config" or "rpc".
//	deviceID is the device being addressed. Use "" for wildcard
//	thingID is the ID of the thing managed by the publisher. Use "" for wildcard
//	name is the event or action name. Use "" for wildcard.
//	clientID is the login ID of the sender. Use "" for subscribe.
func MakeTopic(msgType, deviceID, thingID, name string, clientID string) string {
	if msgType == "" {
		msgType = vocab.MessageTypeEvent
	}
	if deviceID == "" {
		deviceID = "+"
	}
	if thingID == "" {
		thingID = "+"
	}
	if name == "" {
		// sub only. wildcard depends if a clientID follows
		//if clientID == "" {
		//	name = "#"
		//} else {
		name = "+"
		//}
	}
	if clientID == "" {
		clientID = "#"
	}
	topic := fmt.Sprintf("%s/%s/%s/%s/%s", msgType, deviceID, thingID, name, clientID)
	//if clientID != "" {
	//	topic = topic + "/" + clientID
	//}
	return topic
}

// SplitActionTopic separates an action topic into its components
//
// topic is a hiveot mqtt topic. eg: msgType/deviceID/thingID/name/clientID
//
//	msgType is the topic type, eg "action"
//	deviceID is the device or service that handles the topic.
//	thingID is the thing of the topic, or capability for services.
//	name is the action name
//	clientID is the client that publishes the action.
func SplitActionTopic(topic string) (deviceID, thingID, name string, clientID string, err error) {
	var msgType string
	msgType, deviceID, thingID, name, clientID, err = SplitTopic(topic)
	if msgType != "action" {
		err = fmt.Errorf("topic %s is not an action", topic)
		return
	}
	return
}

// SplitTopic separates a topic into its components
//
// topic is a hiveot mqtt topic. eg: msgType/things/deviceID/thingID/name/clientID
//
//	msgType of "things", "rpc" or "_INBOX"
//	deviceID is the device or service that handles the topic.
//	thingID is the thing of the topic, or capability for services.
//	name is the event or action name
//	senderID is the client publishing the request.
func SplitTopic(topic string) (msgType, deviceID, thingID, name string, senderID string, err error) {
	parts := strings.Split(topic, "/")

	// inbox topics are short
	if len(parts) >= 1 && parts[0] == "_INBOX" {
		msgType = parts[0]
		if len(parts) >= 2 {
			deviceID = parts[1]
		}
		return
	}
	if len(parts) < 4 {
		err = errors.New("incomplete topic")
		return
	}
	msgType = parts[0]
	deviceID = parts[1]
	thingID = parts[2]
	name = parts[3]
	if len(parts) > 4 {
		senderID = parts[4]
	}
	return
}
