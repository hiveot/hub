package mqtttransport

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"strings"
)

// MakeTopic creates a mqtt message  topic optionally with wildcards
// This uses the hiveot topic format: {msgType}/{deviceID}/{thingID}/{name}[/{clientID}]
//
//	msgType is the message type: "event", "action", "config" or "rpc".
//	agentID is the device or service being addressed. Use "" for wildcard
//	thingID is the ID of the things managed by the publisher. Use "" for wildcard
//	name is the event or action name. Use "" for wildcard.
//	clientID is the login ID of the sender. Use "" for subscribe.
func MakeTopic(msgType, agentID, thingID, name string, clientID string) string {
	if msgType == "" {
		msgType = vocab.MessageTypeEvent
	}
	if agentID == "" {
		agentID = "+"
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
	topic := fmt.Sprintf("%s/%s/%s/%s/%s", msgType, agentID, thingID, name, clientID)
	//if clientID != "" {
	//	topic = topic + "/" + clientID
	//}
	return topic
}

// SplitTopic separates a topic into its components
//
// topic is a hiveot mqtt topic. eg: msgType/things/deviceID/thingID/name/clientID
//
//	msgType of "things", "rpc" or "_INBOX"
//	agentID is the device or service that is being addressed.
//	thingID is the things of the topic, or capability for services.
//	name is the event or action name
//	senderID is the client publishing the request.
func SplitTopic(topic string) (msgType, agentID, thingID, name string, senderID string, err error) {
	parts := strings.Split(topic, "/")

	// inbox topics are short
	if len(parts) >= 1 && parts[0] == hubclient.MessageTypeINBOX {
		msgType = parts[0]
		if len(parts) >= 2 {
			agentID = parts[1]
		}
		return
	}
	if len(parts) < 4 {
		err = errors.New("incomplete topic")
		return
	}
	msgType = parts[0]
	agentID = parts[1]
	thingID = parts[2]
	name = parts[3]
	if len(parts) > 4 {
		senderID = parts[4]
	}
	return
}
