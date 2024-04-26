// Package digitwin with types and interfaces for using this service
// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.
// Generated 26 Apr 24 14:03 PDT.
package digitwin

import "encoding/json"
import "fmt"
import "github.com/hiveot/hub/runtime/api"
import "github.com/hiveot/hub/lib/things"

// Argument and Response struct for action of Thing 'digitwin'

// ReadThingArgs defines the arguments of the ReadThing function
// Read TD - This returns a single TD document
type ReadThingArgs struct {

	// ThingID Thing ID
	ThingID string `json:"ThingID"`
}

// ReadThingResp defines the response of the ReadThing function
// Read TD - This returns a single TD document
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
	Result []interface{} `json:"Result"`
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

// ReadThing Read TD
// This returns a single TD document
func ReadThing(mt api.IMessageTransport, thingID string) (result string, err error) {
	args := ReadThingArgs{
		ThingID: thingID,
	}
	resp := ReadThingResp{}
	err = mt("digitwin", "readThing", &args, &resp)
	return resp.Result, err
}

// ReadThings Read TDs
// Read a batch of TD documents
func ReadThings(mt api.IMessageTransport, offset int, limit int) (result []interface{}, err error) {
	args := ReadThingsArgs{
		Offset: offset,
		Limit:  limit,
	}
	resp := ReadThingsResp{}
	err = mt("digitwin", "readThings", &args, &resp)
	return resp.Result, err
}

// RemoveThing Remove Thing
// Remove a Thing from the directory and value stores
func RemoveThing(mt api.IMessageTransport, thingID string) (err error) {
	args := RemoveThingArgs{
		ThingID: thingID,
	}
	err = mt("digitwin", "removeThing", &args, nil)
	return err
}

// QueryThings Query Things
// Query things from the directory
func QueryThings(mt api.IMessageTransport, offset int, limit int, query string) (err error) {
	args := QueryThingsArgs{
		Offset: offset,
		Limit:  limit,
		Query:  query,
	}
	err = mt("digitwin", "queryThings", &args, nil)
	return err
}

// IDigitwinService defines the interface of the 'digitwin' service
//
// This defines a method for each of the actions in the TD.
type IDigitwinService interface {

	// QueryThings Query Things
	// Query things from the directory
	QueryThings(args QueryThingsArgs) error

	// ReadThing Read TD
	// This returns a single TD document
	ReadThing(args ReadThingArgs) (ReadThingResp, error)

	// ReadThings Read TDs
	// Read a batch of TD documents
	ReadThings(args ReadThingsArgs) (ReadThingsResp, error)

	// RemoveThing Remove Thing
	// Remove a Thing from the directory and value stores
	RemoveThing(args RemoveThingArgs) error
}

// HandleMessage handles messages for Thing 'digitwin' to be passed to the implementing service
//
// This unmarshals the request payload into a args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func HandleMessage(msg *things.ThingMessage, svc IDigitwinService) (reply []byte, err error) {
	switch msg.Key {
	case "readThing":
		args := ReadThingArgs{}
		err = json.Unmarshal(msg.Data, &args)
		resp, err := svc.ReadThing(args)
		reply, err = json.Marshal(resp)
		return reply, err
	case "readThings":
		args := ReadThingsArgs{}
		err = json.Unmarshal(msg.Data, &args)
		resp, err := svc.ReadThings(args)
		reply, err = json.Marshal(resp)
		return reply, err
	case "removeThing":
		args := RemoveThingArgs{}
		err = json.Unmarshal(msg.Data, &args)
		err := svc.RemoveThing(args)
		return nil, err
	case "queryThings":
		args := QueryThingsArgs{}
		err = json.Unmarshal(msg.Data, &args)
		err := svc.QueryThings(args)
		return nil, err
	}
	return nil, fmt.Errorf("unknown request method '%s'", msg.Key)
}
