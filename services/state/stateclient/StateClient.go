package stateclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/services/state/stateapi"
)

// StateClient is a marshaller for service messages using a provided hub connection.
// This uses the default serializer to marshal and unmarshal messages.
type StateClient struct {
	// ID of the service that handles the requests
	agentID string
	// State storage capability
	thingID string
	// thingID of the state service
	serviceID string
	// Connection to the hub
	hc hubclient.IHubClient
}

// Delete removes the record with the given key.
func (cl *StateClient) Delete(key string) error {

	req := stateapi.DeleteArgs{Key: key}
	err := cl.hc.Rpc(cl.serviceID, stateapi.DeleteMethod, &req, nil)
	return err
}

// Get reads and unmarshals the record with the given key.
// If the key doesn't exist this returns an empty record.
func (cl *StateClient) Get(key string, record interface{}) (found bool, err error) {

	req := stateapi.GetArgs{Key: key}
	resp := stateapi.GetResp{}
	err = cl.hc.Rpc(cl.serviceID, stateapi.GetMethod, &req, &resp)
	if err != nil {
		return false, err
	}
	if resp.Found {
		err = ser.Unmarshal([]byte(resp.Value), record)
	}
	return resp.Found, err
}

// GetMultiple reads multiple records with the given keys.
// This marshalling and unmarshalling is up to the caller.
func (cl *StateClient) GetMultiple(keys []string) (values map[string]string, err error) {

	req := stateapi.GetMultipleArgs{Keys: keys}
	resp := stateapi.GetMultipleResp{}
	err = cl.hc.Rpc(cl.serviceID, stateapi.GetMultipleMethod, &req, &resp)
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
	req := stateapi.SetArgs{Key: key, Value: string(value)}
	err = cl.hc.Rpc(cl.serviceID, stateapi.SetMethod, &req, nil)
	return err
}

// SetMultiple writes multiple record
func (cl *StateClient) SetMultiple(kv map[string]string) error {
	req := stateapi.SetMultipleArgs{KV: kv}
	err := cl.hc.Rpc(cl.serviceID, stateapi.SetMultipleMethod, &req, nil)
	return err
}

// NewStateClient returns a client to access state
//
//	hc is the hub client connection to use.
func NewStateClient(hc hubclient.IHubClient) *StateClient {
	agentID := stateapi.ServiceName
	cl := StateClient{
		hc:        hc,
		agentID:   agentID,
		thingID:   stateapi.StorageThingID,
		serviceID: things.MakeDigiTwinThingID(agentID, stateapi.StorageThingID),
	}
	return &cl
}
