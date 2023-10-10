package hubclient

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/ser"
	"reflect"
)

// CapabilityHandler defines a handler for RPC request
type CapabilityHandler struct {
	// Arguments type or nil if no arguments
	ArgsType any
	// Response type or nil if no response parameters
	RespType any
	// Method that handles the request
	// The method takes the arguments in the ArgsType struct and replies
	// with a type holding the response parameters.
	Method reflect.Value
}

// HandleMessage unmarshal a request message parameters, passes it to the associated method,
// and marshals the result.
// The request argument can be passed by value or reference.
// Intended to remove most boilerplate from handling and dispatching requests.
func (ch *CapabilityHandler) HandleMessage(payload []byte) (respData []byte, err error) {

	args := ch.ArgsType
	resp := ch.RespType

	// magic spells found at: https://github.com/a8m/reflect-examples#call-function-with-list-of-arguments-and-validate-return-values
	// and here: https://stackoverflow.com/questions/45679408/unmarshal-json-to-reflected-struct
	// First determine the type of argument of the method and whether it is passed by value or reference
	// this handler only support a single argument that has to be a struct by value or reference
	t := ch.Method.Type()
	argv := make([]reflect.Value, t.NumIn())
	argType := t.In(0)
	passByRef := (argType.Kind() == reflect.Ptr)

	if args != nil {
		t1 := reflect.TypeOf(args)
		n1 := reflect.New(t1) // pointer to a new zero value of type
		n1El := n1.Elem()
		err = json.Unmarshal(payload, n1El.Addr().Interface())

		if passByRef {
			argv[0] = reflect.ValueOf(n1El.Addr().Interface())
		} else {
			argv[0] = reflect.ValueOf(n1El.Interface())
		}
	}
	resValues := ch.Method.Call(argv)
	var errResp interface{}
	if len(resValues) == 1 {
		// only returns an error value
		resp = nil
		v0 := resValues[0]
		errResp = v0.Interface()
	} else if len(resValues) == 2 {
		// returns a struct result to be marshalled and an error value
		v0 := resValues[0]
		resp = v0.Interface()
		v1 := resValues[1]
		errResp = v1.Interface()
	} else {
		return nil, fmt.Errorf("unexpected result")
	}
	if errResp == nil {
		err = nil
	} else {
		err = errResp.(error)
	}

	if err == nil {
		// marshal the response if arguments are given
		if resp != nil {
			respData, err = ser.Marshal(resp)
		}
	}
	return respData, err
}

// SubRPCCapability is a helper to easily subscribe capability methods with their handler.
//
//	capID is the capability to register
//	capMap maps method names to their handler
//	hc is the service agent connection to the message bus
func SubRPCCapability(capID string, capMap map[string]CapabilityHandler, hc IHubClient) (ISubscription, error) {
	// subscribe to the capability with our own handler.
	// the handler invokes the method registered with the capability map,
	// after unmarshalling the request argument.
	sub, err := hc.SubRPCRequest(capID, func(msg *RequestMessage) error {
		var err error
		var respData []byte

		capHandler, found := capMap[msg.Name]
		if !found {
			return fmt.Errorf("method '%s' not part of capability '%s'", msg.Name, capID)
		}
		respData, err = capHandler.HandleMessage(msg.Payload)
		if respData == nil {
			err = msg.SendAck()
		} else {
			err = msg.SendReply(respData, err)
		}
		return err
	})
	return sub, err
}
