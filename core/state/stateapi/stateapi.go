package stateapi

// ServiceName defines the default state service agent ID
const ServiceName = "state"

// StorageCap identifies the capability to store state
const StorageCap = "store"

// DeleteMethod deletes a record from the store
const DeleteMethod = "delete"

type DeleteArgs struct {
	Key string `json:"key"`
}

// GetMethod reads a record from the store
const GetMethod = "get"

type GetArgs struct {
	Key string `json:"key"`
}

type GetResp struct {
	// The returned key or "" if record wasnt found
	Key string `json:"key"`
	// Flag, the record was found (true)
	Found bool `json:"found"`
	// Data, the raw data of the record
	Value []byte `json:"value"`
}

// GetMultipleMethod reads multiple records from the store
const GetMultipleMethod = "getMultiple"

type GetMultipleArgs struct {
	Keys []string `json:"keys"`
}

type GetMultipleResp struct {
	// Key-values that were found
	KV map[string][]byte `json:"kv"`
}

// SetMethod writes a record to the store
const SetMethod = "set"

type SetArgs struct {
	Key   string `json:"key"`
	Value []byte `json:"value"`
}

// SetMultipleMethod writes multiple records to the store
const SetMultipleMethod = "setMultiple"

type SetMultipleArgs struct {
	KV map[string][]byte `json:"kv"`
}
