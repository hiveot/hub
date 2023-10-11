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
// and marshals the result. Intended to remove boilerplate from handling and serving requests.
// The method argument and result can be a struct value or reference.
//
// Supported method types are:
//
//   - func(type1) (type2,error)
//   - func(type1) (error)
//   - func(type1) ()
//   - func() (type,error)
//   - func() (error)
//   - func() ()
//
// where type1 and type2 can be a struct or native type, or a pointer to a struct or native type.
func HandleRequestMessage(method interface{}, payload []byte) (respData []byte, err error) {

	// magic spells found at: https://github.com/a8m/reflect-examples#call-function-with-list-of-arguments-and-validate-return-values
	// and here: https://stackoverflow.com/questions/45679408/unmarshal-json-to-reflected-struct
	// First determine the type of argument of the method and whether it is passed by value or reference
	// this handler only support a single argument that has to be a struct by value or reference
	methodValue := reflect.ValueOf(method)
	methodType := methodValue.Type()
	argv := make([]reflect.Value, methodType.NumIn())
	nrArgs := methodType.NumIn()

	if nrArgs == 0 {
		// nothing to do here
	} else if nrArgs == 1 {
		// determine the type of argument, if it is passed by value or reference
		argType := methodType.In(0)
		argIsRef := (argType.Kind() == reflect.Ptr)
		n1 := reflect.New(argType) // pointer to a new zero value of type
		n1El := n1.Elem()
		// n1El now contains the value of the argument type.
		// ? Would it not contain a pointer if passed by value ? apparently not ???
		if argIsRef {
			// n1El is a struct pointer value?
			// for some reason, unmarshall still needs to receive the address of it
			err = json.Unmarshal(payload, n1El.Addr().Interface())
			argv[0] = reflect.ValueOf(n1El.Interface())
		} else {
			// n1El is the value, unmarshal to its address
			err = json.Unmarshal(payload, n1El.Addr().Interface())
			argv[0] = reflect.ValueOf(n1El.Interface())
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
//
//	capID is the capability to register
//	capMap maps method names to their implementation
//	hc is the service agent connection to the message bus
func SubRPCCapability(capID string, capMap map[string]interface{}, hc IHubClient) (ISubscription, error) {
	// subscribe to the capability with our own handler.
	// the handler invokes the method registered with the capability map,
	// after unmarshalling the request argument.
	sub, err := hc.SubRPCRequest(capID, func(msg *RequestMessage) error {
		var err error
		var respData []byte

		capMethod, found := capMap[msg.Name]
		if !found {
			return fmt.Errorf("method '%s' not part of capability '%s'", msg.Name, capID)
		}
		respData, err = HandleRequestMessage(capMethod, msg.Payload)
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
