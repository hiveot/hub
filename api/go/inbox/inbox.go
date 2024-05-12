// Package inbox with types and interfaces for using this service with agent 'digitwin'
// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.
// Generated 11 May 24 20:40 PDT.
package inbox

import "encoding/json"
import "errors"
import "github.com/hiveot/hub/runtime/api"
import "github.com/hiveot/hub/lib/things"

// RawThingID is the raw thingID as used by agents. Digitwin adds the urn:{agent} prefix
const RawThingID = "inbox"
const ThingID = "urn:digitwin:inbox"

// Argument and Response struct for action of Thing 'urn:digitwin:inbox'

const ReadLatestMethod = "readLatest"

// ReadLatestArgs defines the arguments of the readLatest function
// Read latest actions - Read the latest request value of each action of a Thing
type ReadLatestArgs struct {

	// Keys Value Key
	Keys []string `json:"keys"`

	// Since Since
	Since string `json:"since"`

	// ThingID Thing ID
	ThingID string `json:"thingID"`
}

// ReadLatestResp defines the response of the readLatest function
// Read latest actions - Read the latest request value of each action of a Thing
type ReadLatestResp struct {

	// Values map of key:ThingValue objects
	Values interface{} `json:"Values"`
}

// ReadLatest client method - Read latest actions.
// Read the latest request value of each action of a Thing
func ReadLatest(mt api.IMessageTransport, args ReadLatestArgs) (resp ReadLatestResp, stat api.DeliveryStatus, err error) {
	stat, err = mt(nil, "urn:digitwin:inbox", "readLatest", &args, &resp)
	if stat.Error != "" {
		err = errors.New(stat.Error)
	}
	return
}

// IInboxService defines the interface of the 'Inbox' service
//
// This defines a method for each of the actions in the TD.
type IInboxService interface {

	// ReadLatest Read latest actions
	// Read the latest request value of each action of a Thing
	ReadLatest(args ReadLatestArgs) (ReadLatestResp, error)
}

// NewActionHandler returns a server handler for Thing 'urn:digitwin:inbox' actions.
//
// This unmarshals the request payload into an args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func NewActionHandler(svc IInboxService) func(*things.ThingMessage) api.DeliveryStatus {
	return func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		var err error
		switch msg.Key {
		case "readLatest":
			args := ReadLatestArgs{}
			err = json.Unmarshal(msg.Data, &args)
			var resp interface{}
			if err == nil {
				resp, err = svc.ReadLatest(args)
			}
			if resp != nil {
				stat.Reply, _ = json.Marshal(resp)
			}
			stat.Completed(msg, err)
			break
		default:
			err = errors.New("Unknown Method '" + msg.Key + "' of service '" + msg.ThingID + "'")
			stat.Failed(msg, err)
		}
		return stat
	}
}
