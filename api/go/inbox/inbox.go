// Package inbox with types and interfaces for using this service with agent 'digitwin'
// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.
// Generated 16 May 24 14:40 PDT.
package inbox

import "encoding/json"
import "errors"
import "github.com/hiveot/hub/runtime/api"
import "github.com/hiveot/hub/lib/things"
import "github.com/hiveot/hub/lib/hubclient"

// RawThingID is the raw thingID as used by agents. Digitwin adds the urn:{agent} prefix
const RawThingID = "inbox"
const ThingID = "dtw:digitwin:inbox"

// Argument and Response struct for action of Thing 'dtw:digitwin:inbox'

const ReadLatestMethod = "readLatest"

// ReadLatestArgs defines the arguments of the readLatest function
// Read latest actions - Read the latest request value of each action of a Thing
type ReadLatestArgs struct {

	// ThingID ID of the Thing to read
	ThingID string `json:"thingID"`

	// Keys The action keys to read or empty to read all latest values
	Keys []string `json:"keys,omitEmpty"`

	// Since Only return values updated since
	Since string `json:"since,omitEmpty"`
}

// ReadLatestResp defines the response of the readLatest function
// Read latest actions - Read the latest request value of each action of a Thing
type ReadLatestResp struct {

	// ThingValues map of key:ThingValue objects
	ThingValues things.ThingMessageMap `json:"thingValues,omitEmpty"`
}

// ReadLatest client method - Read latest actions.
// Read the latest request value of each action of a Thing
func ReadLatest(hc hubclient.IHubClient, args ReadLatestArgs) (resp ReadLatestResp, err error) {
	err = hc.Rpc("dtw:digitwin:inbox", "readLatest", &args, &resp)
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
		stat.Completed(msg, nil)
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
		default:
			err = errors.New("Unknown Method '" + msg.Key + "' of service '" + msg.ThingID + "'")
			stat.Failed(msg, err)
		}
		if resp != nil {
			stat.Reply, _ = json.Marshal(resp)
		}
		if err != nil {
			stat.Error = err.Error()
		}
		return stat
	}
}