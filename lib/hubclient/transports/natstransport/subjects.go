package natstransport

import (
	"errors"
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"strings"
)

// MakeSubject creates a nats subject optionally with nats wildcards
// This uses the hiveot nats subject format: {msgType}.{pubID}.{thingID}.name}.{clientID}
//
//	msgType is the message type: "event", "action", "config" or "rpc".
//	agentID is the device or service being addressed. Use "" for wildcard
//	thingID is the ID of the things managed by the publisher. Use "" for wildcard
//	name is the event or action name. Use "" for wildcard.
//	thingID is the ID of the things managed by the publisher. Use "" for wildcard
//	name is the event or action name. Use "" for wildcard.
//	clientID is the sender's loginID. Required when publishing.
func MakeSubject(msgType, agentID, thingID, name string, clientID string) string {
	if msgType == "" {
		msgType = vocab.MessageTypeEvent
	}
	if agentID == "" {
		agentID = "*"
	}
	if thingID == "" {
		thingID = "*" // nats uses *
	}
	if name == "" {
		//if clientID == "" {
		//	name = ">"
		//} else {
		name = "*"
		//}
	}
	if clientID == "" {
		clientID = ">"
	}
	subj := fmt.Sprintf("%s.%s.%s.%s.%s", msgType, agentID, thingID, name, clientID)
	//if clientID != "" {
	//	subj = subj + "." + clientID
	//}
	return subj
}

// SplitSubject separates a subject into its components
//
// subject is a hiveot nats subject. eg: msgType.publisherID.thingID.type.name.senderID
//
//	msgType of things or services
//	agentID is the device or service that handles the subject.
//	thingID is the things of the subject, or capability for services.
//	name is the event or action name
func SplitSubject(subject string) (msgType, agentID, thingID, name string, senderID string, err error) {
	parts := strings.Split(subject, ".")
	if len(parts) < 4 {
		err = errors.New("incomplete subject")
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
