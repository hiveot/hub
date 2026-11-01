package internal

import (
	"errors"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/launcher"
)

// NewHandleAdminRequest returns an agent handler for Thing 'admin' requests.
//
// This unmarshalls the request payload into an args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func NewHandleAdminRequest(svc launcher.ILauncherService) func(msg *msg.RequestMessage, c api.IConnection) *msg.ResponseMessage {
	return func(msg *msg.RequestMessage, c api.IConnection) *msg.ResponseMessage {
		var output any
		var err error
		switch msg.Name {
		case "startPlugin":
			var args string
			err = utils.DecodeAsObject(msg.Input, &args)
			if err == nil {
				output, err = svc.StartPlugin(args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
		case "stopAllPlugins":
			var args bool
			err = utils.DecodeAsObject(msg.Input, &args)
			if err == nil {
				err = svc.StopAllPlugins(args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
		case "stopPlugin":
			var args string
			err = utils.DecodeAsObject(msg.Input, &args)
			if err == nil {
				output, err = svc.StopPlugin(args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
		case "listPlugins":
			var args bool
			err = utils.DecodeAsObject(msg.Input, &args)
			if err == nil {
				output, err = svc.ListPlugins(args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
		case "startAllPlugins":
			if err == nil {
				err = svc.StartAllPlugins()
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
		default:
			err = errors.New("Unknown Method '" + msg.Name + "' of service '" + msg.ThingID + "'")
		}
		return msg.CreateResponse(output, err)
	}
}
