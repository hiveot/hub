package hubclient

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
//	pubID is the publisher of the subject.
//	thingID is the thing of the subject.
//	stype is the subject type, eg event or action.
//	name is the event or action name
func SplitSubject(subject string) (pubID, thingID, stype, name string, err error) {
	parts := strings.Split(subject, ".")
	if len(parts) < 5 {
		err = errors.New("incomplete subject")
		return
	}
	pubID = parts[1]
	thingID = parts[2]
	stype = parts[3]
	name = parts[4]
	return
}
