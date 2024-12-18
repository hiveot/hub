// Package digitwin with types and interfaces for using this service with agent 'digitwin'
// DO NOT EDIT. This file is auto generated by tdd2api. Any changes will be overwritten.
// Generated 19 Nov 24 19:05 PST.
package digitwin

import "errors"
import "github.com/hiveot/hub/lib/utils"
import "github.com/hiveot/hub/lib/hubclient"

// DirectoryAgentID is the account ID of the agent managing the Thing.
const DirectoryAgentID = "digitwin"

// DirectoryServiceID is the thingID of the device/service as used by agents.
// Agents use this to publish events and subscribe to actions
const DirectoryServiceID = "directory"

// DirectoryDThingID is the Digitwin thingID as used by consumers. Digitwin adds the dtw:{agent} prefix to the serviceID
// Consumers use this to publish actions and subscribe to events
const DirectoryDThingID = "dtw:digitwin:directory"

// Thing names
const (
	DirectoryEventThingUpdated = "thingUpdated"
	DirectoryEventThingRemoved = "thingRemoved"
	DirectoryActionUpdateTD    = "updateTD"
	DirectoryActionReadTD      = "readTD"
	DirectoryActionReadAllTDs  = "readAllTDs"
	DirectoryActionRemoveTD    = "removeTD"
)

//--- Argument and Response struct for action of Thing 'dtw:digitwin:directory' ---

const DirectoryReadAllTDsMethod = "readAllTDs"

// DirectoryReadAllTDsArgs defines the arguments of the readAllTDs function
// Read all TDs - Read a batch of TD documents
type DirectoryReadAllTDsArgs struct {

	// Limit with Limit
	//
	// Maximum number of documents to return
	Limit int `json:"limit,omitempty"`

	// Offset with Offset
	//
	// Start index in the list of TD documents
	Offset int `json:"offset,omitempty"`
}

const DirectoryReadTDMethod = "readTD"

const DirectoryRemoveTDMethod = "removeTD"

const DirectoryUpdateTDMethod = "updateTD"

// DirectoryReadAllTDs client method - Read all TDs.
// Read a batch of TD documents
func DirectoryReadAllTDs(hc hubclient.IConsumerClient, limit int, offset int) (tDlist []string, err error) {
	var args = DirectoryReadAllTDsArgs{limit, offset}
	err = hc.Rpc(DirectoryDThingID, DirectoryReadAllTDsMethod, &args, &tDlist)
	return
}

// DirectoryReadTD client method - Read TD.
// Return a JSON encoded TD document
func DirectoryReadTD(hc hubclient.IConsumerClient, thingID string) (tD string, err error) {

	err = hc.Rpc(DirectoryDThingID, DirectoryReadTDMethod, &thingID, &tD)
	return
}

// DirectoryRemoveTD client method - Remove TD.
// Remove a digital twin from the directory
func DirectoryRemoveTD(hc hubclient.IConsumerClient, dThingID string) (err error) {

	err = hc.Rpc(DirectoryDThingID, DirectoryRemoveTDMethod, &dThingID, nil)
	return
}

// DirectoryUpdateTD client method - Update TD.
// Update the Thing TD in the directory. For use by agents only.
func DirectoryUpdateTD(hc hubclient.IConsumerClient, tD string) (err error) {

	err = hc.Rpc(DirectoryDThingID, DirectoryUpdateTDMethod, &tD, nil)
	return
}

// IDirectoryService defines the interface of the 'Directory' service
//
// This defines a method for each of the actions in the TD.
type IDirectoryService interface {

	// ReadAllTDs Read all TDs
	// Read a batch of TD documents
	// This returns a list of JSON encoded TD documents
	ReadAllTDs(senderID string, args DirectoryReadAllTDsArgs) (tDlist []string, err error)

	// ReadTD Read TD
	// Return a JSON encoded TD document
	// This returns a a JSON encoded Thing Description Document
	ReadTD(senderID string, thingID string) (tD string, err error)

	// RemoveTD Remove TD
	// Remove a digital twin from the directory
	RemoveTD(senderID string, dThingID string) error

	// UpdateTD Update TD
	// Update the Thing TD in the directory. For use by agents only.
	UpdateTD(senderID string, tD string) error
}

// NewHandleDirectoryAction returns a server handler for Thing 'dtw:digitwin:directory' actions.
//
// This unmarshalls the request payload into an args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func NewHandleDirectoryAction(svc IDirectoryService) func(msg *hubclient.ThingMessage) hubclient.RequestStatus {
	return func(msg *hubclient.ThingMessage) (stat hubclient.RequestStatus) {
		var err error
		stat.Completed(msg, nil, nil)
		var output any
		switch msg.Name {
		case "readAllTDs":
			args := DirectoryReadAllTDsArgs{}
			err = utils.DecodeAsObject(msg.Data, &args)
			if err == nil {
				output, err = svc.ReadAllTDs(msg.SenderID, args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
			break
		case "removeTD":
			var args string
			err = utils.DecodeAsObject(msg.Data, &args)
			if err == nil {
				err = svc.RemoveTD(msg.SenderID, args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
			break
		case "updateTD":
			var args string
			err = utils.DecodeAsObject(msg.Data, &args)
			if err == nil {
				err = svc.UpdateTD(msg.SenderID, args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
			break
		case "readTD":
			var args string
			err = utils.DecodeAsObject(msg.Data, &args)
			if err == nil {
				output, err = svc.ReadTD(msg.SenderID, args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
			break
		default:
			err = errors.New("Unknown Method '" + msg.Name + "' of service '" + msg.ThingID + "'")
		}
		stat.Completed(msg, output, err)
		return stat
	}
}

// DirectoryTD contains the raw TD of this service for publication to the Hub
const DirectoryTD = `{"actions":{"readAllTDs":{"@type":"ht:function","description":"Read a batch of TD documents","title":"Read all TDs","idempotent":true,"input":{"readOnly":false,"type":"object","properties":{"limit":{"title":"Limit","description":"Maximum number of documents to return","default":100,"readOnly":false,"type":"integer","minimum":1},"offset":{"title":"Offset","description":"Start index in the list of TD documents","default":0,"readOnly":false,"type":"integer"}}},"output":{"title":"TD list","description":"List of JSON encoded TD documents","readOnly":false,"type":"array","items":{"readOnly":false,"type":"string"}},"safe":true},"readTD":{"@type":"ht:function","description":"Return a JSON encoded TD document","title":"Read TD","idempotent":true,"input":{"title":"Thing ID","description":"Digital Twin Thing ID of the Thing to read","readOnly":false,"type":"string"},"output":{"title":"TD","description":"A JSON encoded Thing Description Document","readOnly":false,"type":"string"},"safe":true},"removeTD":{"@type":"ht:function","description":"Remove a digital twin from the directory","title":"Remove TD","idempotent":true,"input":{"title":"dThing ID","description":"Digital Twin Thing ID of the Thing to remove","readOnly":false,"type":"string"},"allow":["admin"]},"updateTD":{"@type":"ht:function","description":"Update the Thing TD in the directory. For use by agents only.","title":"Update TD","idempotent":true,"input":{"title":"TD","description":"Device TD document in JSON format","readOnly":false,"type":"string"},"allow":["agent","admin"]}},"@context":["https://www.w3.org/2022/wot/td/v1.1",{"ht":"https://www.hiveot.net/vocab/v0.1"}],"@type":"Service","created":"2024-04-21T17:00:00.000Z","deny":["none"],"description":"HiveOT digital twin directory service","events":{"thingRemoved":{"description":"A Thing TD was removed from the directory","title":"Thing Removed","data":{"title":"Thing ID","description":"ID of the digital twin Thing that was removed","readOnly":false,"type":"string"}},"thingUpdated":{"description":"A digital twin Thing TD was updated in the directory","title":"Thing Updated","data":{"title":"TD","description":"JSON encoded TD of the digital twin Thing","readOnly":false,"type":"string"}}},"id":"directory","modified":"2024-04-21T17:00:00.000Z","security":["bearer"],"securityDefinitions":{"bearer":{"description":"HTTP protocol authentication","scheme":"bearer","name":"authentication","alg":"es256","format":"jwt","in":"header"}},"title":"DigiTwin Directory Service","support":"https://www.github.com/hiveot/hub"}`
