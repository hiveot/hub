package api

// IProtocolBinding is the interface implemented by all protocol bindings
type IProtocolBinding interface {
	Start() error

	Stop()
}
