package internal

import (
	"errors"
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/launcher"
)

// HandleRequest handles service requests
func (svc *LauncherServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var output any

	if req.ThingID != svc.GetID() && req.ThingID != launcher.LauncherServiceCellType {
		return svc.ForwardRequest(req, replyTo)
	}

	slog.Info("HandleRequest", slog.String("name", req.Name), slog.String("sender", req.SenderID))
	switch req.Name {
	case launcher.StartPluginAction:
		var args string
		err = utils.DecodeAsObject(req.Input, &args)
		if err == nil {
			output, err = svc.StartPlugin(args)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}
	case launcher.StopAllPluginsAction:
		var args bool
		err = utils.DecodeAsObject(req.Input, &args)
		if err == nil {
			err = svc.StopAllPlugins(args)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}
	case launcher.StopPluginAction:
		var args string
		err = utils.DecodeAsObject(req.Input, &args)
		if err == nil {
			output, err = svc.StopPlugin(args)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}
	case launcher.ListPluginsAction:
		var args bool
		err = utils.DecodeAsObject(req.Input, &args)
		if err == nil {
			output, err = svc.ListPlugins(args)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}
	case launcher.StartAllPluginsAction:
		if err == nil {
			err = svc.StartAllPlugins()
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}
	default:
		err = errors.New("Unknown Method '" + req.Name + "' of service '" + req.ThingID + "'")
	}
	resp := req.CreateResponse(output, err)
	return replyTo(resp)
}
