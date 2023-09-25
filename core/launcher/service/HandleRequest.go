package service

import (
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/launcher"
	"github.com/hiveot/hub/lib/ser"
	"log/slog"
)

// HandleRequest handle incoming RPC requests for managing clients
func (svc *LauncherService) HandleRequest(msg *hubclient.RequestMessage) error {
	slog.Info("HandleRequest", slog.String("actionID", msg.ActionID))

	// TODO: double-check the caller is an admin or svc
	switch msg.ActionID {
	case launcher.LauncherListRPC:
		req := launcher.LauncherListReq{}
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
	case launcher.LauncherStartServiceRPC:
		req := launcher.LauncherStartServiceReq{}
		err := ser.Unmarshal(msg.Payload, &req)
		if err != nil {
			return err
		}
		serviceInfo, err := svc.StartService(req.Name)
		if err == nil {
			resp := launcher.LauncherStartServiceResp{ServiceInfo: serviceInfo}
			reply, _ := ser.Marshal(&resp)
			err = msg.SendReply(reply, nil)
		}
		return err
	case launcher.LauncherStartAllRPC:
		err := svc.StartAll()
		return err
	case launcher.LauncherStopServiceRPC:
		req := launcher.LauncherStopServiceReq{}
		err := ser.Unmarshal(msg.Payload, &req)
		if err != nil {
			return err
		}
		serviceInfo, err := svc.StopService(req.Name)
		if err == nil {
			resp := launcher.LauncherStopServiceResp{ServiceInfo: serviceInfo}
			reply, _ := ser.Marshal(&resp)
			err = msg.SendReply(reply, nil)
		}
		return err
	case launcher.LauncherStopAllRPC:
		err := svc.StopAll()
		if err == nil {
			err = msg.SendAck()
		}
		return err
	}
	return nil
}
