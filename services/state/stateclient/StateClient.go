package stateclient

import (
	"encoding/json"
	"github.com/hiveot/hub/services/state/stateapi"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
)

// StateClient is a the friendly client for service messages using a provided hub connection.
type StateClient struct {
	// dThingID digital twin service ID of the state management
	dThingID string
	// Connection to the hub
	hc transports.IClientConnection
}

// Delete removes the record with the given key.
func (cl *StateClient) Delete(key string) error {

	req := stateapi.DeleteArgs{Key: key}
	err := cl.hc.InvokeAction(cl.dThingID, stateapi.DeleteMethod, &req, nil)
	return err
}

// Get reads  the record with the given key.
// If the key doesn't exist this returns an empty record.
func (cl *StateClient) Get(key string, record interface{}) (found bool, err error) {

	req := stateapi.GetArgs{Key: key}
	resp := stateapi.GetResp{}
	err = cl.hc.InvokeAction(cl.dThingID, stateapi.GetMethod, &req, &resp)
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

	req := stateapi.GetMultipleArgs{Keys: keys}
	resp := stateapi.GetMultipleResp{}
	err = cl.hc.InvokeAction(cl.dThingID, stateapi.GetMultipleMethod, &req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.KV, err
}

// Set serializes and stores a record by the given key
func (cl *StateClient) Set(key string, record interface{}) error {
	data, _ := json.Marshal(record)
	req := stateapi.SetArgs{Key: key, Value: string(data)}
	err := cl.hc.InvokeAction(cl.dThingID, stateapi.SetMethod, &req, nil)
	return err
}

// SetMultiple writes multiple serialized records
func (cl *StateClient) SetMultiple(kv map[string]string) error {
	req := stateapi.SetMultipleArgs{KV: kv}
	err := cl.hc.InvokeAction(cl.dThingID, stateapi.SetMultipleMethod, &req, nil)
	return err
}

// NewStateClient returns a client to access state.
//
// This assumes the agentID used to access the service is: stateapi.StateAgentID.
//
//	hc is the hub client connection to use.
//	agentID is the instance name of the state agent. Use "" for default.
func NewStateClient(hc transports.IClientConnection) *StateClient {
	agentID := stateapi.AgentID
	cl := StateClient{
		hc:       hc,
		dThingID: td.MakeDigiTwinThingID(agentID, stateapi.StorageServiceID),
	}
	return &cl
}
