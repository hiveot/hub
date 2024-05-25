// Package directory with types and interfaces for using this service with agent 'digitwin'
// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.
// Generated 24 May 24 13:34 PDT.
package directory

import "encoding/json"
import "errors"
import "github.com/hiveot/hub/runtime/api"
import "github.com/hiveot/hub/lib/things"
import "github.com/hiveot/hub/lib/hubclient"

// AgentID is the connection ID of the agent managing the Thing.
const AgentID = "digitwin"

// ServiceID is the internal thingID of the device/service as used by agents.
// Agents use this to publish events and subscribe to actions
const ServiceID = "directory"

// DThingID is the Digitwin thingID as used by agents. Digitwin adds the dtw:{agent} prefix to the serviceID
// Consumers use this to publish actions and subscribe to events
const DThingID = "dtw:digitwin:directory"

// Argument and Response struct for action of Thing 'dtw:digitwin:directory'

const ReadTDMethod = "readTD"

// ReadTDArgs defines the arguments of the readTD function
// Read TD - This returns a JSON encoded TD document
type ReadTDArgs struct {

	// ThingID Digital Twin Thing ID of the Thing to read
	ThingID string `json:"thingID"`
}

// ReadTDResp defines the response of the readTD function
// Read TD - This returns a JSON encoded TD document
type ReadTDResp struct {

	// Output A JSON encoded Thing Description Document
	Output string `json:"output"`
}

const ReadTDsMethod = "readTDs"

// ReadTDsArgs defines the arguments of the readTDs function
// Read TDs - Read a batch of TD documents
type ReadTDsArgs struct {

	// Offset Offset
	Offset int `json:"offset,omitempty"`

	// Limit Limit
	Limit int `json:"limit,omitempty"`
}

// ReadTDsResp defines the response of the readTDs function
// Read TDs - Read a batch of TD documents
type ReadTDsResp struct {

	// Output List of JSON encoded TD documents
	Output []string `json:"output"`
}

const RemoveTDMethod = "removeTD"

// RemoveTDArgs defines the arguments of the removeTD function
// Remove TD - Remove a Thing TD document from the digital twin directory and value stores
type RemoveTDArgs struct {

	// ThingID Digital Twin ThingID of the Thing to remove
	ThingID string `json:"thingID,omitempty"`
}

const QueryTDsMethod = "queryTDs"

// QueryTDsArgs defines the arguments of the queryTDs function
// Query TDs - Query Thing TD documents from the directory
type QueryTDsArgs struct {

	// Query Query expression (tbd)
	Query string `json:"query,omitempty"`

	// Offset Number of results to skip
	Offset int `json:"offset,omitempty"`

	// Limit Maximum number of entries to return
	Limit int `json:"limit,omitempty"`
}

// QueryTDsResp defines the response of the queryTDs function
// Query TDs - Query Thing TD documents from the directory
type QueryTDsResp struct {

	// Output List of JSON encoded TD documents
	Output []string `json:"output"`
}

// QueryTDs client method - Query TDs.
// Query Thing TD documents from the directory
func QueryTDs(hc hubclient.IHubClient, args QueryTDsArgs) (resp QueryTDsResp, err error) {
	err = hc.Rpc(DThingID, "queryTDs", &args, &resp)
	return
}

// ReadTD client method - Read TD.
// This returns a JSON encoded TD document
func ReadTD(hc hubclient.IHubClient, args ReadTDArgs) (resp ReadTDResp, err error) {
	err = hc.Rpc(DThingID, "readTD", &args, &resp)
	return
}

// ReadTDs client method - Read TDs.
// Read a batch of TD documents
func ReadTDs(hc hubclient.IHubClient, args ReadTDsArgs) (resp ReadTDsResp, err error) {
	err = hc.Rpc(DThingID, "readTDs", &args, &resp)
	return
}

// RemoveTD client method - Remove TD.
// Remove a Thing TD document from the digital twin directory and value stores
func RemoveTD(hc hubclient.IHubClient, args RemoveTDArgs) (err error) {
	err = hc.Rpc(DThingID, "removeTD", &args, nil)
	return
}

// IDirectoryService defines the interface of the 'Directory' service
//
// This defines a method for each of the actions in the TD.
type IDirectoryService interface {

	// QueryTDs Query TDs
	// Query Thing TD documents from the directory
	QueryTDs(args QueryTDsArgs) (QueryTDsResp, error)

	// ReadTD Read TD
	// This returns a JSON encoded TD document
	ReadTD(args ReadTDArgs) (ReadTDResp, error)

	// ReadTDs Read TDs
	// Read a batch of TD documents
	ReadTDs(args ReadTDsArgs) (ReadTDsResp, error)

	// RemoveTD Remove TD
	// Remove a Thing TD document from the digital twin directory and value stores
	RemoveTD(args RemoveTDArgs) error
}

// NewActionHandler returns a server handler for Thing 'dtw:digitwin:directory' actions.
//
// This unmarshals the request payload into an args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func NewActionHandler(svc IDirectoryService) func(*things.ThingMessage) api.DeliveryStatus {
	return func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		var err error
		var resp interface{}
		switch msg.Key {
		case "readTD":
			args := ReadTDArgs{}
			err = json.Unmarshal(msg.Data, &args)
			if err == nil {
				resp, err = svc.ReadTD(args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
			break
		case "readTDs":
			args := ReadTDsArgs{}
			err = json.Unmarshal(msg.Data, &args)
			if err == nil {
				resp, err = svc.ReadTDs(args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
			break
		case "removeTD":
			args := RemoveTDArgs{}
			err = json.Unmarshal(msg.Data, &args)
			if err == nil {
				err = svc.RemoveTD(args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
			break
		case "queryTDs":
			args := QueryTDsArgs{}
			err = json.Unmarshal(msg.Data, &args)
			if err == nil {
				resp, err = svc.QueryTDs(args)
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
