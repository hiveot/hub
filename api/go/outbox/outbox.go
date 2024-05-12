// Package outbox with types and interfaces for using this service with agent 'digitwin'
// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.
// Generated 11 May 24 20:40 PDT.
package outbox

import "encoding/json"
import "errors"
import "github.com/hiveot/hub/runtime/api"
import "github.com/hiveot/hub/lib/things"

// RawThingID is the raw thingID as used by agents. Digitwin adds the urn:{agent} prefix
const RawThingID = "outbox"
const ThingID = "urn:digitwin:outbox"

// Argument and Response struct for action of Thing 'urn:digitwin:outbox'

const ReadLatestMethod = "readLatest"

// ReadLatestArgs defines the arguments of the readLatest function
// Read Latest - Read the latest value(s) of a Thing
type ReadLatestArgs struct {

	// ThingID Thing ID
	ThingID string `json:"thingID"`

	// Keys Value Key
	Keys []string `json:"keys"`

	// Since Since
	Since string `json:"since"`
}

// ReadLatestResp defines the response of the readLatest function
// Read Latest - Read the latest value(s) of a Thing
type ReadLatestResp struct {

	// Values map of key:ThingValue objects
	Values interface{} `json:"Values"`
}

const RemoveValueMethod = "removeValue"

// RemoveValueArgs defines the arguments of the removeValue function
// Remove Thing Value - Remove a value
type RemoveValueArgs struct {

	// MessageID Message ID
	MessageID string `json:"messageID"`
}

// ReadLatest client method - Read Latest.
// Read the latest value(s) of a Thing
func ReadLatest(mt api.IMessageTransport, args ReadLatestArgs) (resp ReadLatestResp, stat api.DeliveryStatus, err error) {
	stat, err = mt(nil, "urn:digitwin:outbox", "readLatest", &args, &resp)
	if stat.Error != "" {
		err = errors.New(stat.Error)
	}
	return
}

// RemoveValue client method - Remove Thing Value.
// Remove a value
func RemoveValue(mt api.IMessageTransport, args RemoveValueArgs) (stat api.DeliveryStatus, err error) {
	stat, err = mt(nil, "urn:digitwin:outbox", "removeValue", &args, nil)
	if stat.Error != "" {
		err = errors.New(stat.Error)
	}
	return
}

// IOutboxService defines the interface of the 'Outbox' service
//
// This defines a method for each of the actions in the TD.
type IOutboxService interface {

	// ReadLatest Read Latest
	// Read the latest value(s) of a Thing
	ReadLatest(args ReadLatestArgs) (ReadLatestResp, error)

	// RemoveValue Remove Thing Value
	// Remove a value
	RemoveValue(args RemoveValueArgs) error
}

// NewActionHandler returns a server handler for Thing 'urn:digitwin:outbox' actions.
//
// This unmarshals the request payload into an args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func NewActionHandler(svc IOutboxService) func(*things.ThingMessage) api.DeliveryStatus {
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
		case "removeValue":
			args := RemoveValueArgs{}
			err = json.Unmarshal(msg.Data, &args)
			if err == nil {
				err = svc.RemoveValue(args)
			}
			break
		default:
			err = errors.New("Unknown Method '" + msg.Key + "' of service '" + msg.ThingID + "'")
			stat.Failed(msg, err)
		}
		return stat
	}
}
