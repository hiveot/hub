package stateclient

import (
	"github.com/hiveot/hub/core/state/stateapi"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
)

// StateClient is a marshaller for service messages using a provided hub connection.
// This uses the default serializer to marshal and unmarshal messages.
type StateClient struct {
	// ID of the service that handles the requests
	agentID string
	// State storage capability
	capID string
	// Connection to the hub
	hc hubclient.IHubClient
}

// Delete removes the record with the given key.
func (cl *StateClient) Delete(key string) error {

	req := stateapi.DeleteArgs{Key: key}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, stateapi.DeleteMethod, &req, nil)
	return err
}

// Get reads and unmarshals the record with the given key.
// If the key doesn't exist this returns an empty record.
func (cl *StateClient) Get(key string, record interface{}) (found bool, err error) {

	req := stateapi.GetArgs{Key: key}
	resp := stateapi.GetResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, stateapi.GetMethod, &req, &resp)
	if err != nil {
		return false, err
	}
	if resp.Found {
		err = ser.Unmarshal(resp.Value, record)
	}
	return resp.Found, err
}

// GetMultiple reads multiple records with the given keys.
// This marshalling and unmarshalling is up to the caller.
func (cl *StateClient) GetMultiple(keys []string) (values map[string][]byte, err error) {

	req := stateapi.GetMultipleArgs{Keys: keys}
	resp := stateapi.GetMultipleResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, stateapi.GetMultipleMethod, &req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.KV, err
}

// Set marshals and writes a record
func (cl *StateClient) Set(key string, record interface{}) error {
	value, err := ser.Marshal(record)
	if err != nil {
		return err
	}
	req := stateapi.SetArgs{Key: key, Value: value}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, stateapi.SetMethod, &req, nil)
	return err
}

// SetMultiple writes multiple record
func (cl *StateClient) SetMultiple(kv map[string][]byte) error {
	req := stateapi.SetMultipleArgs{KV: kv}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, stateapi.SetMultipleMethod, &req, nil)
	return err
}

// NewStateClient returns a client to access state
//
//	hc is the hub client connection to use.
func NewStateClient(hc hubclient.IHubClient) *StateClient {
	agentID := stateapi.ServiceName
	cl := StateClient{
		hc:      hc,
		agentID: agentID,
		capID:   stateapi.StorageCap,
	}
	return &cl
}
