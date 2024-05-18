package stateapi

// ServiceName defines the default state service agent ID
const ServiceName = "state"

// StorageThingID identifies the ThingID of the capability to store state
const StorageThingID = "store"

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
	Value string `json:"value"`
}

type GetMultipleArgs struct {
	Keys []string `json:"keys"`
}

type GetMultipleResp struct {
	// Key-values that were found
	KV map[string]string `json:"kv"`
}

type SetArgs struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SetMultipleArgs struct {
	KV map[string]string `json:"kv"`
}
