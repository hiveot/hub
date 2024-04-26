package digitwin

import "github.com/hiveot/hub/runtime/api"

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

	// Limit
	Limit int `json:"Limit"`

	// Offset
	Offset int `json:"Offset"`
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

	// Offset
	Offset int `json:"Offset"`

	// Limit Limit
	Limit int `json:"Limit"`

	// Query Query
	Query string `json:"Query"`
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
func QueryThings(mt api.IMessageTransport, limit int, query string, offset int) (err error) {
	args := QueryThingsArgs{
		Limit:  limit,
		Query:  query,
		Offset: offset,
	}
	err = mt("digitwin", "queryThings", &args, nil)
	return err
}
