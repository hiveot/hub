// Package outbox with types and interfaces for using this service
// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.
// Generated 08 May 24 18:55 PDT.
package outbox

import "encoding/json"
import "fmt"
import "github.com/hiveot/hub/runtime/api"
import "github.com/hiveot/hub/lib/things"

// the raw thingID as used by agents. Digitwin adds the urn:{agent} prefix
const RawThingID = "outbox"
const ThingID = "urn:digitwin:outbox"

// Argument and Response struct for action of Thing 'outbox'

// ReadLatestArgs defines the arguments of the ReadLatest function
// Read Latest - Read the latest value(s) of a Thing
type ReadLatestArgs struct {

	// ThingID Thing ID
	ThingID string `json:"ThingID"`

	// Keys Value Key
	Keys []string `json:"Keys"`

	// Since Since
	Since string `json:"Since"`
}

// ReadLatestResp defines the response of the ReadLatest function
// Read Latest - Read the latest value(s) of a Thing
type ReadLatestResp struct {

	// Values map of key:ThingValue objects
	Values interface{} `json:"Values"`
}

// RemoveValueArgs defines the arguments of the RemoveValue function
// Remove Thing Value - Remove a value
type RemoveValueArgs struct {

	// MessageID Message ID
	MessageID string `json:"MessageID"`
}

// RemoveValue Remove Thing Value
// Remove a value
func RemoveValue(mt api.IMessageTransport, messageID string) (err error) {
	args := RemoveValueArgs{
		MessageID: messageID,
	}
	err = mt("outbox", "removeValue", &args, nil)
	return err
}

// ReadLatest Read Latest
// Read the latest value(s) of a Thing
func ReadLatest(mt api.IMessageTransport, thingID string, keys []string, since string) (Values interface{}, err error) {
	args := ReadLatestArgs{
		ThingID: thingID,
		Keys:    keys,
		Since:   since,
	}
	resp := ReadLatestResp{}
	err = mt("outbox", "readLatest", &args, &resp)
	return resp.Values, err
}

// IOutboxService defines the interface of the 'outbox' service
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

// NewActionHandler returns a handler for Thing 'outbox' actions to be passed to the implementing service
//
// This unmarshals the request payload into a args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func NewActionHandler(svc IOutboxService) func(*things.ThingMessage) api.DeliveryStatus {
	return func(msg *things.ThingMessage) api.DeliveryStatus {
		var err = fmt.Errorf("unknown action '%s'", msg.Key)
		var status = api.DeliveryFailed
		res := api.DeliveryStatus{}
		switch msg.Key {
		case "readLatest":
			args := ReadLatestArgs{}
			var resp interface{}
			err = json.Unmarshal(msg.Data, &args)
			resp, err = svc.ReadLatest(args)
			if err == nil {
				res.Reply, err = json.Marshal(resp)
				status = api.DeliveryCompleted
			}
			break
		case "removeValue":
			args := RemoveValueArgs{}
			err = json.Unmarshal(msg.Data, &args)
			err = svc.RemoveValue(args)
			if err == nil {
				status = api.DeliveryCompleted
			}
			break
		}
		res.Status = status
		if err != nil {
			res.Error = err.Error()
		}
		return res
	}
}
