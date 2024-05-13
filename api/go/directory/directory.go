// Package directory with types and interfaces for using this service with agent 'digitwin'
// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.
// Generated 13 May 24 15:19 PDT.
package directory

import "encoding/json"
import "errors"
import "github.com/hiveot/hub/runtime/api"
import "github.com/hiveot/hub/lib/things"

// RawThingID is the raw thingID as used by agents. Digitwin adds the urn:{agent} prefix
const RawThingID = "directory"
const ThingID = "dtw:digitwin:directory"

// Argument and Response struct for action of Thing 'dtw:digitwin:directory'

const ReadThingMethod = "readThing"

// ReadThingArgs defines the arguments of the readThing function
// Read TD - This returns a JSON encoded TD document
type ReadThingArgs struct {

	// ThingID Digital Twin Thing ID of the Thing to read
	ThingID string `json:"thingID"`
}

// ReadThingResp defines the response of the readThing function
// Read TD - This returns a JSON encoded TD document
type ReadThingResp struct {

	// Output A JSON encoded Thing Description Document
	Output string `json:"output"`
}

const ReadThingsMethod = "readThings"

// ReadThingsArgs defines the arguments of the readThings function
// Read TDs - Read a batch of TD documents
type ReadThingsArgs struct {

	// Offset Offset
	Offset int `json:"offset,omitEmpty"`

	// Limit Limit
	Limit int `json:"limit,omitEmpty"`
}

// ReadThingsResp defines the response of the readThings function
// Read TDs - Read a batch of TD documents
type ReadThingsResp struct {

	// Output List of JSON encoded TD documents
	Output []string `json:"output"`
}

const RemoveThingMethod = "removeThing"

// RemoveThingArgs defines the arguments of the removeThing function
// Remove Thing - Remove a Thing from the digital twin directory and value stores
type RemoveThingArgs struct {

	// ThingID Digital Twin ThingID of the Thing to remove
	ThingID string `json:"thingID,omitEmpty"`
}

const QueryThingsMethod = "queryThings"

// QueryThingsArgs defines the arguments of the queryThings function
// Query Things - Query things from the directory
type QueryThingsArgs struct {

	// Query Query expression (tbd)
	Query string `json:"query,omitEmpty"`

	// Offset Number of results to skip
	Offset int `json:"offset,omitEmpty"`

	// Limit Maximum number of entries to return
	Limit int `json:"limit,omitEmpty"`
}

// QueryThingsResp defines the response of the queryThings function
// Query Things - Query things from the directory
type QueryThingsResp struct {

	// Output List of JSON encoded TD documents
	Output []string `json:"output"`
}

// ReadThing client method - Read TD.
// This returns a JSON encoded TD document
func ReadThing(mt api.IMessageTransport, args ReadThingArgs) (resp ReadThingResp, stat api.DeliveryStatus, err error) {
	stat, err = mt(nil, "dtw:digitwin:directory", "readThing", &args, &resp)
	if stat.Error != "" {
		err = errors.New(stat.Error)
	}
	return
}

// ReadThings client method - Read TDs.
// Read a batch of TD documents
func ReadThings(mt api.IMessageTransport, args ReadThingsArgs) (resp ReadThingsResp, stat api.DeliveryStatus, err error) {
	stat, err = mt(nil, "dtw:digitwin:directory", "readThings", &args, &resp)
	if stat.Error != "" {
		err = errors.New(stat.Error)
	}
	return
}

// RemoveThing client method - Remove Thing.
// Remove a Thing from the digital twin directory and value stores
func RemoveThing(mt api.IMessageTransport, args RemoveThingArgs) (stat api.DeliveryStatus, err error) {
	stat, err = mt(nil, "dtw:digitwin:directory", "removeThing", &args, nil)
	if stat.Error != "" {
		err = errors.New(stat.Error)
	}
	return
}

// QueryThings client method - Query Things.
// Query things from the directory
func QueryThings(mt api.IMessageTransport, args QueryThingsArgs) (resp QueryThingsResp, stat api.DeliveryStatus, err error) {
	stat, err = mt(nil, "dtw:digitwin:directory", "queryThings", &args, &resp)
	if stat.Error != "" {
		err = errors.New(stat.Error)
	}
	return
}

// IDirectoryService defines the interface of the 'Directory' service
//
// This defines a method for each of the actions in the TD.
type IDirectoryService interface {

	// RemoveThing Remove Thing
	// Remove a Thing from the digital twin directory and value stores
	RemoveThing(args RemoveThingArgs) error

	// QueryThings Query Things
	// Query things from the directory
	QueryThings(args QueryThingsArgs) (QueryThingsResp, error)

	// ReadThing Read TD
	// This returns a JSON encoded TD document
	ReadThing(args ReadThingArgs) (ReadThingResp, error)

	// ReadThings Read TDs
	// Read a batch of TD documents
	ReadThings(args ReadThingsArgs) (ReadThingsResp, error)
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
		stat.Completed(msg, nil)
		switch msg.Key {
		case "removeThing":
			args := RemoveThingArgs{}
			err = json.Unmarshal(msg.Data, &args)
			if err == nil {
				err = svc.RemoveThing(args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
			break
		case "queryThings":
			args := QueryThingsArgs{}
			err = json.Unmarshal(msg.Data, &args)
			if err == nil {
				resp, err = svc.QueryThings(args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
			break
		case "readThing":
			args := ReadThingArgs{}
			err = json.Unmarshal(msg.Data, &args)
			if err == nil {
				resp, err = svc.ReadThing(args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
			break
		case "readThings":
			args := ReadThingsArgs{}
			err = json.Unmarshal(msg.Data, &args)
			if err == nil {
				resp, err = svc.ReadThings(args)
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
