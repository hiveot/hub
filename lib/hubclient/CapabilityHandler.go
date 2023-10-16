package hubclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/ser"
	"log/slog"
	"reflect"
)

// HandleRequestMessage unmarshal a request message parameters, passes it to the associated method,
// and marshals the result. Intended to remove boilerplate from RPC service request handlers.
//
// The first argument is always the clientID of the client invoking the request.
// The second argument and result can be a struct value or reference.
//
// Supported method types are:
//
//   - func(string, type1) (type2,error)
//   - func(string, type1) (error)
//   - func(string, type1) ()
//   - func(string) (type,error)
//   - func(string) (error)
//   - func(string) ()
//
// where type1 and type2 can be a struct or native type, or a pointer to a struct or native type.
func HandleRequestMessage(clientID string, method interface{}, payload []byte) (respData []byte, err error) {

	// magic spells found at: https://github.com/a8m/reflect-examples#call-function-with-list-of-arguments-and-validate-return-values
	// and here: https://stackoverflow.com/questions/45679408/unmarshal-json-to-reflected-struct
	// First determine the type of argument of the method and whether it is passed by value or reference
	// this handler only support a single argument that has to be a struct by value or reference
	methodValue := reflect.ValueOf(method)
	methodType := methodValue.Type()
	argv := make([]reflect.Value, methodType.NumIn())
	nrArgs := methodType.NumIn()

	if nrArgs == 0 {
		// nothing to do here, apparently not interested in the clientID
	} else if nrArgs == 1 {
		// determine the type of argument, if it is passed by value or reference
		argType := methodType.In(0)
		argIsClientID := (argType.Kind() == reflect.String)
		if argIsClientID {
			argv[0] = reflect.ValueOf(clientID)
		}
	} else if nrArgs == 2 {
		// determine the type of argument, if it is passed by value or reference
		argv[0] = reflect.ValueOf(clientID)
		argType := methodType.In(1)
		argIsRef := (argType.Kind() == reflect.Ptr)
		n1 := reflect.New(argType) // pointer to a new zero value of type
		n1El := n1.Elem()
		// n1El now contains the value of the argument type.
		// ? Would it not contain a pointer if passed by value ? apparently not ???
		if argIsRef {
			// n1El is a struct pointer value?
			// for some reason, unmarshall still needs to receive the address of it
			err = json.Unmarshal(payload, n1El.Addr().Interface())
			argv[1] = reflect.ValueOf(n1El.Interface())
		} else {
			// n1El is the value, unmarshal to its address
			err = json.Unmarshal(payload, n1El.Addr().Interface())
			argv[1] = reflect.ValueOf(n1El.Interface())
		}
		if err != nil {
			slog.Error("HandleRequestMessage, failed unmarshal request", "err", err)
		}
	} else {
		return nil, fmt.Errorf("multiple arguments is not supported")
	}
	resValues := methodValue.Call(argv)
	var errResp interface{}
	var resp interface{}
	if len(resValues) == 0 {
		errResp = nil
	} else if len(resValues) == 1 {
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
		return nil,
			errors.New("method has more than 2 result params. This is not supported")
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
// The handler must have signature func(clientID string, args interface{})(interface{},error)
//
//	hc is the agent connection to the message bus
//	capID is the capability to register
//	capMethods maps method names to their implementation
func SubRPCCapability(hc IHubClient, capID string, capMethods map[string]interface{}) (ISubscription, error) {
	// subscribe to the capability with our own handler.
	// the handler invokes the method registered with the capability map,
	// after unmarshalling the request argument.
	sub, err := hc.SubRPCRequest(capID, func(msg *RequestMessage) error {
		var err error
		var respData []byte

		capMethod, found := capMethods[msg.Name]
		if !found {
			return fmt.Errorf("method '%s' not part of capability '%s'", msg.Name, capID)
		}
		respData, err = HandleRequestMessage(msg.ClientID, capMethod, msg.Payload)
		if err == nil {
			if respData == nil {
				err = msg.SendAck()
			} else {
				err = msg.SendReply(respData, err)
			}
		}
		return err
	})
	return sub, err
}
