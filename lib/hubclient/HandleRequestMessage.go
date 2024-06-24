package hubclient

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/ser"
	"reflect"
)

// ServiceContext with context provided to services
//type ServiceContext struct {
//	context.Context
//	// SenderID of the caller
//	SenderID string
//}

// HandleRequestMessage unmarshal a request message parameters, passes it to the associated method,
// and marshals the result. Intended to remove boilerplate from RPC service request handlers.
//
// The first argument can optionally be the senderID of the clientID invoking the request.
// The second argument and results can be a struct value or reference.
//
// Since the arguments are JSON serialized, the wire protocol expects
// a single (type1) struct that holds the parameters. If the handler
// implements multiple arguments, they all receive the same payload if they
// are of the same type.
//
// Supported method types are:
//
//   - func([string,] type1) (type2,error)
//   - func([string,] type1) (error)
//   - func([string,] type1) ()
//   - func([string]) (type,error)
//   - func([string]) (error)
//   - func([string]) ()
//
// where type1 and type2 can be a struct or native type, or a pointer to a struct or native type.
func HandleRequestMessage(senderID string, method interface{}, payload string) (respData string, err error) {

	// magic spells found at: https://github.com/a8m/reflect-examples#call-function-with-list-of-arguments-and-validate-return-values
	// and here: https://stackoverflow.com/questions/45679408/unmarshal-json-to-reflected-struct
	// First determine the type of argument of the method and whether it is passed by value or reference
	methodValue := reflect.ValueOf(method)
	methodType := methodValue.Type()
	argv := make([]reflect.Value, methodType.NumIn())
	nrArgs := methodType.NumIn()

	for i := 0; i < nrArgs; i++ {
		// determine the type of argument, expect the senderID string as first arg
		argType := methodType.In(i)
		argKind := argType.Name()
		argIsSenderID := i == 0 && argKind == "string"
		if argIsSenderID {
			// first argument is the sender clientID
			argv[i] = reflect.ValueOf(senderID)
		} else {
			// the argument is not a senderID string
			argIsRef := (argType.Kind() == reflect.Ptr)
			n1 := reflect.New(argType) // pointer to a new zero value of type
			n1El := n1.Elem()
			// n1El now contains the value of the argument type.
			// ? Would it not contain a pointer if passed by value ? apparently not ???
			if argIsRef {
				// n1El is a struct pointer value?
				// for some reason, unmarshall still needs to receive the address of it
				err = ser.Unmarshal([]byte(payload), n1El.Addr().Interface())
				argv[i] = reflect.ValueOf(n1El.Interface())
			} else {
				// n1El is the value, unmarshal to its address
				err = ser.Unmarshal([]byte(payload), n1El.Addr().Interface())
				argv[i] = reflect.ValueOf(n1El.Interface())
			}
			if err != nil {
				return "", fmt.Errorf("failed unmarshal request: %s", err)
			}
		}

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
		return "",
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
			respJSON, err2 := ser.Marshal(resp)
			err = err2
			respData = string(respJSON)
		}
	}
	return respData, err
}

// RegisterRPCCapability is a helper to easily subscribe capability methods with their handler.
// The handler must have signature func(clientID string, args interface{})(interface{},error)
//
//		hc is the agent connection to the message bus
//		capID is the capability to register
//		capMethods maps method names to their implementation. Supported formats:
//	    1: no context, args or result: func()(error)
//	    2: context, args, and result : func(*ServiceContext,args interface{})(interface{},error)
//	    3: context, args, no result  : func(*ServiceContext,args interface{})(error)
//	    4: context, no args, result  : func(*ServiceContext)(interface{}, error)
//func RegisterRPCCapability(hc IHubTransport, capID string, capMethods map[string]interface{}) (ISubscription, error) {
//	// subscribe to the capability with our own handler.
//	// the handler invokes the method registered with the capability map,
//	// after unmarshalling the request argument.
//	sub, err := hc.SubRequest(capID, func(msg *RequestMessage) error {
//		var err error
//		var respData []byte
//
//		capMethod, found := capMethods[msg.Name]
//		if !found {
//			return fmt.Errorf("method '%s' not part of capability '%s'", msg.Name, capID)
//		}
//		ctx := ServiceContext{
//			Context:  context.Background(),
//			SenderID: msg.SenderID,
//		}
//		respData, err = HandleRequestMessage(ctx, capMethod, msg.Payload)
//		if err == nil {
//			if respData == nil {
//				err = msg.SendAck()
//			} else {
//				err = msg.SendReply(respData, err)
//			}
//		}
//		return err
//	})
//	return sub, err
//}
