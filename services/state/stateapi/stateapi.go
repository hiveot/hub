package stateapi

// AgentID is the connect ID of the agent connecting to the Hub
const AgentID = "state"

// StorageServiceID is the ID of the service as used by the agent
const StorageServiceID = "store"

// Storage methods
const (
	// DeleteMethod deletes a record from the store
	DeleteMethod = "delete"

	// GetMethod reads a record from the store
	GetMethod = "get"

	// GetMultipleMethod reads multiple records from the store
	GetMultipleMethod = "getMultiple"

	// SetMethod writes a record to the store
	SetMethod = "set"

	// SetMultipleMethod writes multiple records to the store
	SetMultipleMethod = "setMultiple"
)

type GetResp struct {
	// The returned key or "" if record wasn't found
	Key string `json:"key"`
	// Flag, the record was found
	Found bool `json:"found"`
	// Data, the json encoded data of the record
	Value string `json:"value"`
}

type GetMultipleArgs []string

type GetMultipleResp map[string]string

type SetArgs struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SetMultipleArgs map[string]string
