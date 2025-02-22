package stateclient

import (
	"encoding/json"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/services/state/stateapi"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
)

// StateClient is  the friendly client for service messages using a provided hub connection.
type StateClient struct {
	// dThingID digital twin service ID of the state management
	dThingID string
	// Thing consumer
	co *messaging.Consumer
}

// Delete removes the record with the given key.
func (cl *StateClient) Delete(key string) error {

	err := cl.co.InvokeAction(cl.dThingID, stateapi.DeleteMethod, key, nil)
	return err
}

// Get reads  the record with the given key.
// If the key doesn't exist this returns an empty record.
func (cl *StateClient) Get(key string, record interface{}) (found bool, err error) {

	resp := stateapi.GetResp{}
	err = cl.co.InvokeAction(cl.dThingID, stateapi.GetMethod, key, &resp)
	if err != nil {
		return false, err
	}
	if resp.Found {
		err = jsoniter.UnmarshalFromString(resp.Value, record)
	}
	return resp.Found, err
}

// GetMultiple reads multiple serialized records with the given keys.
func (cl *StateClient) GetMultiple(keys []string) (values map[string]string, err error) {

	resp := stateapi.GetMultipleResp{}
	err = cl.co.InvokeAction(cl.dThingID, stateapi.GetMultipleMethod, &keys, &resp)
	if err != nil {
		return nil, err
	}
	return resp, err
}

// Set serializes and stores a record by the given key
func (cl *StateClient) Set(key string, record interface{}) error {
	data, _ := json.Marshal(record)
	req := stateapi.SetArgs{Key: key, Value: string(data)}
	err := cl.co.InvokeAction(cl.dThingID, stateapi.SetMethod, &req, nil)

	return err
}

// SetMultiple writes multiple serialized records
func (cl *StateClient) SetMultiple(kv map[string]string) error {
	err := cl.co.InvokeAction(cl.dThingID, stateapi.SetMultipleMethod, kv, nil)
	return err
}

// NewStateClient returns a client to access state.
//
// This assumes the agentID used to access the service is: stateapi.AgentID.
//
//	co is the hub client connection to use.
//	agentID is the instance name of the state agent. Use "" for default.
func NewStateClient(co *messaging.Consumer) *StateClient {
	agentID := stateapi.AgentID
	cl := StateClient{
		co:       co,
		dThingID: td.MakeDigiTwinThingID(agentID, stateapi.StorageServiceID),
	}
	return &cl
}
