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

type DeleteArgs struct {
	Key string `json:"key"`
}

type GetArgs struct {
	Key string `json:"key"`
}

type GetResp struct {
	// The returned key or "" if record wasn't found
	Key string `json:"key"`
	// Flag, the record was found (true)
	Found bool `json:"found"`
	// Data, the raw data of the record
	Value any `json:"value"`
}

type GetMultipleArgs struct {
	Keys []string `json:"keys"`
}

type GetMultipleResp struct {
	// Key-values that were found
	KV map[string]any `json:"kv"`
}

type SetArgs struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type SetMultipleArgs struct {
	KV map[string]any `json:"kv"`
}
