package direct

import (
	"encoding/json"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/router"
)

// DirectTransport acts as client side transport of requests directly to a
// service's message handler.
// Intended for testing service functionality without having to setup an actual
// protocol binding.
//
// The Transport() function implements the api.IMessageTransport interface and
// can be used directly (no pun intended) with service clients.
//
// For example:
// > cl := NewAuthnUserClient( NewDirectTransport("myID", authnSvr.HandleMessage) )
// > token,err := cl.Login("myID", "mypass")
//type DirectTransport struct {
//	// service message handler
//	messageHandler router.MessageHandler
//	// clientID represents the authenticated client, eg agent,service or end-user
//	clientID string
//}
//
//func (dt *DirectTransport) Transport(thingID string, method string, args interface{}, reply interface{}) error {
//	data, _ := json.Marshal(args)
//	tv := things.NewThingMessage(vocab.MessageTypeAction, thingID, method, data, dt.clientID)
//	resp, err := dt.messageHandler(tv)
//	if err == nil {
//		err = json.Unmarshal(resp, &reply)
//	}
//	return err
//}

// NewDirectTransport returns a function to pass messages to a handler.
//
// Intended for testing of clients to transport a request directly to the handler of a service,
// instead of using a protocol binding.
//
// This marshals the request data and builds a ThingMessage object to pass to the handler.
func NewDirectTransport(clientID string, handler router.MessageHandler) func(thingID string, method string, args interface{}, reply interface{}) error {
	return func(thingID string, method string, args interface{}, reply interface{}) error {
		data, _ := json.Marshal(args)
		tv := things.NewThingMessage(vocab.MessageTypeAction, thingID, method, data, clientID)
		resp, err := handler(tv)
		if err == nil && resp != nil {
			err = json.Unmarshal(resp, &reply)
		}
		return err
	}
}
