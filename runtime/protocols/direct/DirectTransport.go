package direct

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/router"
)

// NewDirectTransport returns a function to create a ThingMessage object and pass it to a handler.
//
// Intended for use by clients to encode arguments, transport it to a handler and decode the
// response.
//
// This marshals the request data and builds a ThingMessage object to pass to the handler.
func NewDirectTransport(clientID string, handler router.MessageHandler) func(
	thingID string, method string, args interface{}, reply interface{}) error {

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

// WriteActionMessage is a convenience function to create an action ThingMessage and pass it to
// a handler for routing to its destination.
// This returns the reply data or an error.
func WriteActionMessage(
	thingID string, key string, data []byte, senderID string, handler router.MessageHandler) ([]byte, error) {
	tv := things.NewThingMessage(vocab.MessageTypeAction, thingID, key, data, senderID)
	return handler(tv)
}
