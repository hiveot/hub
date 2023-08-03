package natshubclient

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"strings"
)

// MakeSubject creates a nats subject optionally with nats wildcards
// This uses the hiveot nats subject format: things.{pubID}.{thingID}.{type}.{name}
//
//	pubID is the publisher of the subject. Use "" for wildcard
//	thingID is the ID of the thing managed by the publisher. Use "" for wildcard
//	stype is the subject type: "event" or "action".
//	name is the event or action name. Use "" for wildcard.
func MakeSubject(pubID, thingID, stype, name string) string {
	if pubID == "" {
		pubID = "*"
	}
	if thingID == "" {
		thingID = "*" // nats uses *
	}
	if stype == "" {
		stype = vocab.VocabEventTopic
	}
	if name == "" {
		name = "*" // nats uses *
	}
	subj := fmt.Sprintf("things.%s.%s.%s.%s",
		pubID, thingID, stype, name)
	return subj
}

// SplitSubject separates a subject into its components
//
// subject is a hiveot nats subject. eg: things.publisherID.thingID.type.name
//
//	bindingID is the device or service that handles the subject.
//	thingID is the thing of the subject, or capability for services.
//	stype is the subject type, eg event or action.
//	name is the event or action name
func SplitSubject(subject string) (bindingID, thingID, stype, name string, err error) {
	parts := strings.Split(subject, ".")
	if len(parts) < 5 {
		err = errors.New("incomplete subject")
		return
	}
	bindingID = parts[1]
	thingID = parts[2]
	stype = parts[3]
	name = parts[4]
	return
}

// MakeActionSubject creates a nats subject for submitting actions
// This uses the hiveot nats subject format: things.{bindingID}.{thingID}.action.{name}.{clientID}
//
//	pubID is the publisher of the subject. Use "" for wildcard
//	thingID is the ID of the thing managed by the publisher. Use "" for wildcard
//	name is the event or action name. Use "" for wildcard.
//	clientID is the loginID of the user submitting the action
func MakeActionSubject(pubID, thingID, name string, clientID string) string {
	if pubID == "" {
		pubID = "*"
	}
	if thingID == "" {
		thingID = "*" // nats uses *
	}
	if name == "" {
		name = "*" // nats uses *
	}
	if clientID == "" {
		clientID = "*"
	}
	subj := fmt.Sprintf("things.%s.%s.%s.%s.%s",
		pubID, thingID, vocab.VocabActionTopic, name, clientID)
	return subj
}

// SplitActionSubject separates a subject into its components
//
// subject is a hiveot nats subject. eg: things.bindingID.thingID.stype.name.clientID
//
//	bindingID is the device or service that handles the subject.
//	thingID is the thing of the subject, or capability for services.
//	stype is the subject type, eg event or action.
//	name is the action name
//	clientID is the client that publishes the action. This identifies the publisher.
func SplitActionSubject(subject string) (bindingID, thingID, name string, clientID string, err error) {
	parts := strings.Split(subject, ".")
	if len(parts) < 6 {
		err = errors.New("incomplete subject")
		return
	}
	stype := parts[3]
	if stype != "action" {
		err = fmt.Errorf("subject %s is not an action", subject)
		return
	}
	bindingID = parts[1]
	thingID = parts[2]
	name = parts[4]
	clientID = parts[5]
	return
}
