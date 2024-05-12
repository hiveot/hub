// Package directory with types and interfaces for using this service with agent 'digitwin'
// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.
// Generated 11 May 24 20:40 PDT.
package directory

import "encoding/json"
import "errors"
import "github.com/hiveot/hub/runtime/api"
import "github.com/hiveot/hub/lib/things"

// RawThingID is the raw thingID as used by agents. Digitwin adds the urn:{agent} prefix
const RawThingID = "directory"
const ThingID = "urn:digitwin:directory"

// Argument and Response struct for action of Thing 'urn:digitwin:directory'

const ReadThingMethod = "readThing"

// ReadThingArgs defines the arguments of the readThing function
// Read TD - This returns a JSON encoded TD document
type ReadThingArgs struct {

	// ThingID Thing ID
	ThingID string `json:"thingID"`
}

// ReadThingResp defines the response of the readThing function
// Read TD - This returns a JSON encoded TD document
type ReadThingResp struct {

	// Output TDD
	Output string `json:"output"`
}

const ReadThingsMethod = "readThings"

// ReadThingsArgs defines the arguments of the readThings function
// Read TDs - Read a batch of TD documents
type ReadThingsArgs struct {

	// Offset
	Offset int `json:"offset"`

	// Limit
	Limit int `json:"limit"`
}

// ReadThingsResp defines the response of the readThings function
// Read TDs - Read a batch of TD documents
type ReadThingsResp struct {

	// Output TD list
	Output []string `json:"output"`
}

const RemoveThingMethod = "removeThing"

// RemoveThingArgs defines the arguments of the removeThing function
// Remove Thing - Remove a Thing from the directory and value stores
type RemoveThingArgs struct {

	// ThingID thingID
	ThingID string `json:"thingID"`
}

const QueryThingsMethod = "queryThings"

// QueryThingsArgs defines the arguments of the queryThings function
// Query Things - Query things from the directory
type QueryThingsArgs struct {

	// Query Query Things
	Query string `json:"query"`

	// Offset Result offset
	Offset int `json:"offset"`

	// Limit Max entries
	Limit int `json:"limit"`
}

// QueryThingsResp defines the response of the queryThings function
// Query Things - Query things from the directory
type QueryThingsResp struct {

	// Output TD list
	Output []string `json:"output"`
}

// ReadThing client method - Read TD.
// This returns a JSON encoded TD document
func ReadThing(mt api.IMessageTransport, args ReadThingArgs) (resp ReadThingResp, stat api.DeliveryStatus, err error) {
	stat, err = mt(nil, "urn:digitwin:directory", "readThing", &args, &resp)
	if stat.Error != "" {
		err = errors.New(stat.Error)
	}
	return
}

// ReadThings client method - Read TDs.
// Read a batch of TD documents
func ReadThings(mt api.IMessageTransport, args ReadThingsArgs) (resp ReadThingsResp, stat api.DeliveryStatus, err error) {
	stat, err = mt(nil, "urn:digitwin:directory", "readThings", &args, &resp)
	if stat.Error != "" {
		err = errors.New(stat.Error)
	}
	return
}

// RemoveThing client method - Remove Thing.
// Remove a Thing from the directory and value stores
func RemoveThing(mt api.IMessageTransport, args RemoveThingArgs) (stat api.DeliveryStatus, err error) {
	stat, err = mt(nil, "urn:digitwin:directory", "removeThing", &args, nil)
	if stat.Error != "" {
		err = errors.New(stat.Error)
	}
	return
}

// QueryThings client method - Query Things.
// Query things from the directory
func QueryThings(mt api.IMessageTransport, args QueryThingsArgs) (resp QueryThingsResp, stat api.DeliveryStatus, err error) {
	stat, err = mt(nil, "urn:digitwin:directory", "queryThings", &args, &resp)
	if stat.Error != "" {
		err = errors.New(stat.Error)
	}
	return
}

// IDirectoryService defines the interface of the 'Directory' service
//
// This defines a method for each of the actions in the TD.
type IDirectoryService interface {

	// ReadThing Read TD
	// This returns a JSON encoded TD document
	ReadThing(args ReadThingArgs) (ReadThingResp, error)

	// ReadThings Read TDs
	// Read a batch of TD documents
	ReadThings(args ReadThingsArgs) (ReadThingsResp, error)

	// RemoveThing Remove Thing
	// Remove a Thing from the directory and value stores
	RemoveThing(args RemoveThingArgs) error

	// QueryThings Query Things
	// Query things from the directory
	QueryThings(args QueryThingsArgs) (QueryThingsResp, error)
}

// NewActionHandler returns a server handler for Thing 'urn:digitwin:directory' actions.
//
// This unmarshals the request payload into an args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func NewActionHandler(svc IDirectoryService) func(*things.ThingMessage) api.DeliveryStatus {
	return func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		var err error
		switch msg.Key {
		case "readThings":
			args := ReadThingsArgs{}
			err = json.Unmarshal(msg.Data, &args)
			var resp interface{}
			if err == nil {
				resp, err = svc.ReadThings(args)
			}
			if resp != nil {
				stat.Reply, _ = json.Marshal(resp)
			}
			stat.Completed(msg, err)
			break
		case "removeThing":
			args := RemoveThingArgs{}
			err = json.Unmarshal(msg.Data, &args)
			if err == nil {
				err = svc.RemoveThing(args)
			}
			stat.Completed(msg, err)
			break
		case "queryThings":
			args := QueryThingsArgs{}
			err = json.Unmarshal(msg.Data, &args)
			var resp interface{}
			if err == nil {
				resp, err = svc.QueryThings(args)
			}
			if resp != nil {
				stat.Reply, _ = json.Marshal(resp)
			}
			stat.Completed(msg, err)
			break
		case "readThing":
			args := ReadThingArgs{}
			err = json.Unmarshal(msg.Data, &args)
			var resp interface{}
			if err == nil {
				resp, err = svc.ReadThing(args)
			}
			if resp != nil {
				stat.Reply, _ = json.Marshal(resp)
			}
			stat.Completed(msg, err)
			break
		default:
			err = errors.New("Unknown Method '" + msg.Key + "' of service '" + msg.ThingID + "'")
			stat.Failed(msg, err)
		}
		return stat
	}
}
