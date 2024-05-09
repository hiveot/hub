// Package directory with types and interfaces for using this service
// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.
// Generated 08 May 24 18:55 PDT.
package directory

import "encoding/json"
import "fmt"
import "github.com/hiveot/hub/runtime/api"
import "github.com/hiveot/hub/lib/things"

// the raw thingID as used by agents. Digitwin adds the urn:{agent} prefix
const RawThingID = "directory"
const ThingID = "urn:digitwin:directory"

// Argument and Response struct for action of Thing 'directory'

// ReadThingArgs defines the arguments of the ReadThing function
// Read TD - This returns a JSON encoded TD document
type ReadThingArgs struct {

	// ThingID Thing ID
	ThingID string `json:"ThingID"`
}

// ReadThingResp defines the response of the ReadThing function
// Read TD - This returns a JSON encoded TD document
type ReadThingResp struct {

	// Result TDD
	Result string `json:"Result"`
}

// ReadThingsArgs defines the arguments of the ReadThings function
// Read TDs - Read a batch of TD documents
type ReadThingsArgs struct {

	// Offset
	Offset int `json:"Offset"`

	// Limit
	Limit int `json:"Limit"`
}

// ReadThingsResp defines the response of the ReadThings function
// Read TDs - Read a batch of TD documents
type ReadThingsResp struct {

	// Result TD list
	Result []string `json:"Result"`
}

// RemoveThingArgs defines the arguments of the RemoveThing function
// Remove Thing - Remove a Thing from the directory and value stores
type RemoveThingArgs struct {

	// ThingID thingID
	ThingID string `json:"ThingID"`
}

// QueryThingsArgs defines the arguments of the QueryThings function
// Query Things - Query things from the directory
type QueryThingsArgs struct {

	// Query Query Things
	Query string `json:"Query"`

	// Offset Result offset
	Offset int `json:"Offset"`

	// Limit Max entries
	Limit int `json:"Limit"`
}

// QueryThingsResp defines the response of the QueryThings function
// Query Things - Query things from the directory
type QueryThingsResp struct {

	// Result TD list
	Result []string `json:"Result"`
}

// ReadThing Read TD
// This returns a JSON encoded TD document
func ReadThing(mt api.IMessageTransport, thingID string) (result string, err error) {
	args := ReadThingArgs{
		ThingID: thingID,
	}
	resp := ReadThingResp{}
	err = mt("directory", "readThing", &args, &resp)
	return resp.Result, err
}

// ReadThings Read TDs
// Read a batch of TD documents
func ReadThings(mt api.IMessageTransport, offset int, limit int) (result []string, err error) {
	args := ReadThingsArgs{
		Offset: offset,
		Limit:  limit,
	}
	resp := ReadThingsResp{}
	err = mt("directory", "readThings", &args, &resp)
	return resp.Result, err
}

// RemoveThing Remove Thing
// Remove a Thing from the directory and value stores
func RemoveThing(mt api.IMessageTransport, thingID string) (err error) {
	args := RemoveThingArgs{
		ThingID: thingID,
	}
	err = mt("directory", "removeThing", &args, nil)
	return err
}

// QueryThings Query Things
// Query things from the directory
func QueryThings(mt api.IMessageTransport, query string, offset int, limit int) (result []string, err error) {
	args := QueryThingsArgs{
		Query:  query,
		Offset: offset,
		Limit:  limit,
	}
	resp := QueryThingsResp{}
	err = mt("directory", "queryThings", &args, &resp)
	return resp.Result, err
}

// IDirectoryService defines the interface of the 'directory' service
//
// This defines a method for each of the actions in the TD.
type IDirectoryService interface {

	// QueryThings Query Things
	// Query things from the directory
	QueryThings(args QueryThingsArgs) (QueryThingsResp, error)

	// ReadThing Read TD
	// This returns a JSON encoded TD document
	ReadThing(args ReadThingArgs) (ReadThingResp, error)

	// ReadThings Read TDs
	// Read a batch of TD documents
	ReadThings(args ReadThingsArgs) (ReadThingsResp, error)

	// RemoveThing Remove Thing
	// Remove a Thing from the directory and value stores
	RemoveThing(args RemoveThingArgs) error
}

// NewActionHandler returns a handler for Thing 'directory' actions to be passed to the implementing service
//
// This unmarshals the request payload into a args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func NewActionHandler(svc IDirectoryService) func(*things.ThingMessage) api.DeliveryStatus {
	return func(msg *things.ThingMessage) api.DeliveryStatus {
		var err = fmt.Errorf("unknown action '%s'", msg.Key)
		var status = api.DeliveryFailed
		res := api.DeliveryStatus{}
		switch msg.Key {
		case "queryThings":
			args := QueryThingsArgs{}
			var resp interface{}
			err = json.Unmarshal(msg.Data, &args)
			resp, err = svc.QueryThings(args)
			if err == nil {
				res.Reply, err = json.Marshal(resp)
				status = api.DeliveryCompleted
			}
			break
		case "readThing":
			args := ReadThingArgs{}
			var resp interface{}
			err = json.Unmarshal(msg.Data, &args)
			resp, err = svc.ReadThing(args)
			if err == nil {
				res.Reply, err = json.Marshal(resp)
				status = api.DeliveryCompleted
			}
			break
		case "readThings":
			args := ReadThingsArgs{}
			var resp interface{}
			err = json.Unmarshal(msg.Data, &args)
			resp, err = svc.ReadThings(args)
			if err == nil {
				res.Reply, err = json.Marshal(resp)
				status = api.DeliveryCompleted
			}
			break
		case "removeThing":
			args := RemoveThingArgs{}
			err = json.Unmarshal(msg.Data, &args)
			err = svc.RemoveThing(args)
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
