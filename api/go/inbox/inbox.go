// Package inbox with types and interfaces for using this service
// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.
// Generated 08 May 24 18:55 PDT.
package inbox

import "encoding/json"
import "fmt"
import "github.com/hiveot/hub/runtime/api"
import "github.com/hiveot/hub/lib/things"

// the raw thingID as used by agents. Digitwin adds the urn:{agent} prefix
const RawThingID = "inbox"
const ThingID = "urn:digitwin:inbox"

// Argument and Response struct for action of Thing 'inbox'

// ReadLatestArgs defines the arguments of the ReadLatest function
// Read latest actions - Read the latest request value of each action of a Thing
type ReadLatestArgs struct {

	// ThingID Thing ID
	ThingID string `json:"ThingID"`

	// Keys Value Key
	Keys []string `json:"Keys"`

	// Since Since
	Since string `json:"Since"`
}

// ReadLatestResp defines the response of the ReadLatest function
// Read latest actions - Read the latest request value of each action of a Thing
type ReadLatestResp struct {

	// Values map of key:ThingValue objects
	Values interface{} `json:"Values"`
}

// ReadLatest Read latest actions
// Read the latest request value of each action of a Thing
func ReadLatest(mt api.IMessageTransport, keys []string, since string, thingID string) (Values interface{}, err error) {
	args := ReadLatestArgs{
		Keys:    keys,
		Since:   since,
		ThingID: thingID,
	}
	resp := ReadLatestResp{}
	err = mt("inbox", "readLatest", &args, &resp)
	return resp.Values, err
}

// IInboxService defines the interface of the 'inbox' service
//
// This defines a method for each of the actions in the TD.
type IInboxService interface {

	// ReadLatest Read latest actions
	// Read the latest request value of each action of a Thing
	ReadLatest(args ReadLatestArgs) (ReadLatestResp, error)
}

// NewActionHandler returns a handler for Thing 'inbox' actions to be passed to the implementing service
//
// This unmarshals the request payload into a args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func NewActionHandler(svc IInboxService) func(*things.ThingMessage) api.DeliveryStatus {
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
		}
		res.Status = status
		if err != nil {
			res.Error = err.Error()
		}
		return res
	}
}
