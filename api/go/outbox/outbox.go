// Package outbox with types and interfaces for using this service with agent 'digitwin'
// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.
// Generated 24 May 24 13:34 PDT.
package outbox

import "encoding/json"
import "errors"
import "github.com/hiveot/hub/runtime/api"
import "github.com/hiveot/hub/lib/things"
import "github.com/hiveot/hub/lib/hubclient"

// AgentID is the connection ID of the agent managing the Thing.
const AgentID = "digitwin"

// ServiceID is the internal thingID of the device/service as used by agents.
// Agents use this to publish events and subscribe to actions
const ServiceID = "outbox"

// DThingID is the Digitwin thingID as used by agents. Digitwin adds the dtw:{agent} prefix to the serviceID
// Consumers use this to publish actions and subscribe to events
const DThingID = "dtw:digitwin:outbox"

// Argument and Response struct for action of Thing 'dtw:digitwin:outbox'

const ReadLatestMethod = "readLatest"

// ReadLatestArgs defines the arguments of the readLatest function
// Read Latest - Read the latest value(s) of a Thing
type ReadLatestArgs struct {

	// ThingID ID of the Thing to read
	ThingID string `json:"thingID"`

	// Keys The event/property IDs to read or empty to read all latest values
	Keys []string `json:"keys,omitempty"`

	// Since Only return values updated since
	Since string `json:"since,omitempty"`
}

// ReadLatestResp defines the response of the readLatest function
// Read Latest - Read the latest value(s) of a Thing
type ReadLatestResp struct {

	// Values JSON encoded map of key:ThingMessage objects
	Values string `json:"Values,omitempty"`
}

const RemoveValueMethod = "removeValue"

// RemoveValueArgs defines the arguments of the removeValue function
// Remove Thing Value - Remove a value
type RemoveValueArgs struct {

	// MessageID ID of the message to remove
	MessageID string `json:"messageID,omitempty"`
}

// ReadLatest client method - Read Latest.
// Read the latest value(s) of a Thing
func ReadLatest(hc hubclient.IHubClient, args ReadLatestArgs) (resp ReadLatestResp, err error) {
	err = hc.Rpc(DThingID, "readLatest", &args, &resp)
	return
}

// RemoveValue client method - Remove Thing Value.
// Remove a value
func RemoveValue(hc hubclient.IHubClient, args RemoveValueArgs) (err error) {
	err = hc.Rpc(DThingID, "removeValue", &args, nil)
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

// NewActionHandler returns a server handler for Thing 'dtw:digitwin:outbox' actions.
//
// This unmarshals the request payload into an args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func NewActionHandler(svc IOutboxService) func(*things.ThingMessage) api.DeliveryStatus {
	return func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		var err error
		var resp interface{}
		switch msg.Key {
		case "readLatest":
			args := ReadLatestArgs{}
			err = json.Unmarshal(msg.Data, &args)
			if err == nil {
				resp, err = svc.ReadLatest(args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
			break
		case "removeValue":
			args := RemoveValueArgs{}
			err = json.Unmarshal(msg.Data, &args)
			if err == nil {
				err = svc.RemoveValue(args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
			break
		default:
			err = errors.New("Unknown Method '" + msg.Key + "' of service '" + msg.ThingID + "'")
			stat.Failed(msg, err)
		}
		if resp != nil {
			stat.Reply, _ = json.Marshal(resp)
		}
		stat.Completed(msg, err)
		return stat
	}
}
