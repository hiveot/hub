// Package digitwin with types and interfaces for using this service with agent 'digitwin'
// DO NOT EDIT. This file is auto generated by tdd2api. Any changes will be overwritten.
// Generated 09 Jun 24 10:18 PDT.
package digitwin

import "encoding/json"
import "errors"
import "github.com/hiveot/hub/lib/things"
import "github.com/hiveot/hub/lib/hubclient"

// InboxAgentID is the connection ID of the agent managing the Thing.
const InboxAgentID = "digitwin"

// InboxServiceID is the internal thingID of the device/service as used by agents.
// Agents use this to publish events and subscribe to actions
const InboxServiceID = "inbox"

// InboxDThingID is the Digitwin thingID as used by agents. Digitwin adds the dtw:{agent} prefix to the serviceID
// Consumers use this to publish actions and subscribe to events
const InboxDThingID = "dtw:digitwin:inbox"

//--- Argument and Response struct for action of Thing 'dtw:digitwin:inbox' ---

const InboxReadLatestMethod = "readLatest"

// InboxReadLatestArgs defines the arguments of the readLatest function
// Read latest actions - Read the latest request value of each action of a Thing
type InboxReadLatestArgs struct {

	// Keys with Value Key
	//
	// The action keys to read or empty to read all latest values
	Keys []string `json:"keys,omitempty"`

	// Since with Since
	//
	// Only return values updated since
	Since string `json:"since,omitempty"`

	// ThingID with Thing ID
	//
	// ID of the Thing to read
	ThingID string `json:"thingID,omitempty"`
}

// InboxClient client for talking to the 'dtw:digitwin:inbox' service
type InboxClient struct {
	dThingID string
	hc       hubclient.IHubClient
}

// ReadLatest client method - Read latest actions.
// Read the latest request value of each action of a Thing
func (svc *InboxClient) ReadLatest(args InboxReadLatestArgs) (valueMap things.ThingMessageMap, err error) {
	err = svc.hc.Rpc(svc.dThingID, InboxReadLatestMethod, &args, &valueMap)
	return
}

// NewInboxClient creates a new client for invoking DigiTwin Inbox methods.
func NewInboxClient(hc hubclient.IHubClient) *InboxClient {
	cl := InboxClient{
		hc:       hc,
		dThingID: "dtw:digitwin:inbox",
	}
	return &cl
}

// IInboxService defines the interface of the 'Inbox' service
//
// This defines a method for each of the actions in the TD.
type IInboxService interface {

	// ReadLatest Read latest actions
	// Read the latest request value of each action of a Thing
	// This returns a map of key:ThingMessage objects
	ReadLatest(senderID string, args InboxReadLatestArgs) (valueMap things.ThingMessageMap, err error)
}

// NewInboxHandler returns a server handler for Thing 'dtw:digitwin:inbox' actions.
//
// This unmarshalls the request payload into an args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func NewInboxHandler(svc IInboxService) func(*things.ThingMessage) hubclient.DeliveryStatus {
	return func(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
		var err error
		var resp interface{}
		var senderID = msg.SenderID
		switch msg.Key {
		case "readLatest":
			args := InboxReadLatestArgs{}
			err = msg.Unmarshal(&args)
			if err == nil {
				resp, err = svc.ReadLatest(senderID, args)
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