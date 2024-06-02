// Package inbox with types and interfaces for using this service with agent 'digitwin'
// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.
// Generated 01 Jun 24 20:18 PDT.
package inbox

import "encoding/json"
import "errors"
import "github.com/hiveot/hub/runtime/api"
import "github.com/hiveot/hub/lib/things"
import "github.com/hiveot/hub/lib/hubclient"

// AgentID is the connection ID of the agent managing the Thing.
const AgentID = "digitwin"

// ServiceID is the internal thingID of the device/service as used by agents.
// Agents use this to publish events and subscribe to actions
const ServiceID = "inbox"

// DThingID is the Digitwin thingID as used by agents. Digitwin adds the dtw:{agent} prefix to the serviceID
// Consumers use this to publish actions and subscribe to events
const DThingID = "dtw:digitwin:inbox"

// Argument and Response struct for action of Thing 'dtw:digitwin:inbox'

const ReadLatestMethod = "readLatest"

// ReadLatestArgs defines the arguments of the readLatest function
// Read latest actions - Read the latest request value of each action of a Thing
type ReadLatestArgs struct {

	// Since Only return values updated since
	Since string `json:"since,omitempty"`

	// ThingID ID of the Thing to read
	ThingID string `json:"thingID"`

	// Keys The action keys to read or empty to read all latest values
	Keys []string `json:"keys,omitempty"`
}

// ReadLatestResp defines the response of the readLatest function
// Read latest actions - Read the latest request value of each action of a Thing
type ReadLatestResp struct {

	// ThingValues map of key:ThingValue objects
	ThingValues things.ThingMessageMap `json:"thingValues,omitempty"`
}

// ReadLatest client method - Read latest actions.
// Read the latest request value of each action of a Thing
func ReadLatest(hc hubclient.IHubClient, args ReadLatestArgs) (resp ReadLatestResp, err error) {
	err = hc.Rpc(DThingID, "readLatest", &args, &resp)
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

// NewActionHandler returns a server handler for Thing 'dtw:digitwin:inbox' actions.
//
// This unmarshals the request payload into an args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func NewActionHandler(svc IInboxService) func(*things.ThingMessage) api.DeliveryStatus {
	return func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		var err error
		var resp interface{}
		switch msg.Key {
		case "readLatest":
			args := ReadLatestArgs{}
			err = msg.Unmarshal(&args)
			if err == nil {
				resp, err = svc.ReadLatest(args)
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
