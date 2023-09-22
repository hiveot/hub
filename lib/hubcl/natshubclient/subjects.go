package natshubclient

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"strings"
)

// MakeSubject creates a nats subject optionally with nats wildcards
// This uses the hiveot nats subject format: {msgType}.{pubID}.{thingID}.name}[.{clientID}]
//
//	msgType is the message type: "event", "action", "config" or "rpc".
//	deviceID is the publisher of the subject. Use "" for wildcard
//	thingID is the ID of the thing managed by the publisher. Use "" for wildcard
//	name is the event or action name. Use "" for wildcard.
//	thingID is the ID of the thing managed by the publisher. Use "" for wildcard
//	name is the event or action name. Use "" for wildcard.
//	clientID is required for publishing action and rpc requests.
func MakeSubject(msgType, deviceID, thingID, name string, clientID string) string {
	if msgType == "" {
		msgType = vocab.MessageTypeEvent
	}
	if deviceID == "" {
		deviceID = "*"
	}
	if thingID == "" {
		thingID = "*" // nats uses *
	}
	if name == "" {
		if clientID == "" {
			name = ">"
		} else {
			name = "*"
		}
	}
	subj := fmt.Sprintf("%s.%s.%s.%s", msgType, deviceID, thingID, name)
	if clientID != "" {
		subj = subj + "." + clientID
	}
	return subj
}

// SplitSubject separates a subject into its components
//
// subject is a hiveot nats subject. eg: msgType.publisherID.thingID.type.name.senderID
//
//	msgType of things or services
//	deviceID is the device or service that handles the subject.
//	thingID is the thing of the subject, or capability for services.
//	name is the event or action name
func SplitSubject(subject string) (msgType, deviceID, thingID, name string, senderID string, err error) {
	parts := strings.Split(subject, ".")
	if len(parts) < 4 {
		err = errors.New("incomplete subject")
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
