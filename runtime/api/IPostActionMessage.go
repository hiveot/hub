package api

// IPostActionMessage defines the function for posting action messages to the hub and services
// Intended for clients of the runtime to pass encoded RPC messages to the runtime protocol binding
// This method encodes the request arguments and decodes the response into the reply struct
type IPostActionMessage func(thingID string, method string, args interface{}, reply interface{}) error
