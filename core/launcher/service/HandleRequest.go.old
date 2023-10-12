package service

import (
	launcher "github.com/hiveot/hub/core/launcher"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"log/slog"
)

// HandleRequest handle incoming RPC requests for managing clients
func (svc *LauncherService) HandleRequest(msg *hubclient.RequestMessage) error {
	slog.Info("HandleRequest", slog.String("actionID", msg.Name))

	// TODO: double-check the caller is an admin or svc
	switch msg.Name {
	case launcher.LauncherListReq:
		req := launcher.LauncherListArgs{}
		err := ser.Unmarshal(msg.Payload, &req)
		if err != nil {
			return err
		}
		serviceInfoList, err := svc.List(req.OnlyRunning)
		if err == nil {
			resp := launcher.LauncherListResp{ServiceInfoList: serviceInfoList}
			reply, _ := ser.Marshal(&resp)
			err = msg.SendReply(reply, nil)
		}
		return err
	case launcher.LauncherStartPluginReq:
		req := launcher.LauncherStartPluginArgs{}
		err := ser.Unmarshal(msg.Payload, &req)
		if err != nil {
			return err
		}
		serviceInfo, err := svc.StartPlugin(req.Name)
		if err == nil {
			resp := launcher.LauncherStartPluginResp{ServiceInfo: serviceInfo}
			reply, _ := ser.Marshal(&resp)
			err = msg.SendReply(reply, nil)
		}
		return err
	case launcher.LauncherStartAllPluginsReq:
		err := svc.StartAllPlugins()
		return err
	case launcher.LauncherStopPluginReq:
		req := launcher.LauncherStopPluginArgs{}
		err := ser.Unmarshal(msg.Payload, &req)
		if err != nil {
			return err
		}
		serviceInfo, err := svc.StopPlugin(req.Name)
		if err == nil {
			resp := launcher.LauncherStopPluginResp{ServiceInfo: serviceInfo}
			reply, _ := ser.Marshal(&resp)
			err = msg.SendReply(reply, nil)
		}
		return err
	case launcher.LauncherStopAllPluginsReq:
		err := svc.StopAllPlugins()
		if err == nil {
			err = msg.SendAck()
		}
		return err
	}
	return nil
}
