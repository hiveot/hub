package selfsigned

import (
	"github.com/hiveot/hub/core/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"log/slog"
)

// HandleRequest handle incoming RPC requests for managing clients
func (svc *SelfSignedCertsService) HandleRequest(action *hubclient.RequestMessage) error {
	slog.Info("HandleRequest", slog.String("actionID", action.ActionID))

	// TODO: double-check the caller is an admin or svc
	switch action.ActionID {
	case certs.CreateDeviceCertReq:
		req := certs.CreateDeviceCertArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		certPEM, caCertPEM, err := svc.CreateDeviceCert(req.DeviceID, req.PubKeyPEM, req.ValidityDays)
		if err == nil {
			resp := certs.CreateCertResp{CertPEM: certPEM, CaCertPEM: caCertPEM}
			reply, _ := ser.Marshal(&resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case certs.CreateServiceCertReq:
		req := certs.CreateServiceCertArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		certPEM, caCertPEM, err := svc.CreateServiceCert(
			req.ServiceID, req.PubKeyPEM, req.Names, req.ValidityDays)
		if err == nil {
			resp := certs.CreateCertResp{CertPEM: certPEM, CaCertPEM: caCertPEM}
			reply, _ := ser.Marshal(&resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case certs.CreateUserCertReq:
		req := certs.CreateUserCertArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		certPEM, caCertPEM, err := svc.CreateUserCert(
			req.UserID, req.PubKeyPEM, req.ValidityDays)
		if err == nil {
			resp := certs.CreateCertResp{CertPEM: certPEM, CaCertPEM: caCertPEM}
			reply, _ := ser.Marshal(&resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case certs.VerifyCertReq:
		req := certs.VerifyCertArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.VerifyCert(req.ClientID, req.CertPEM)
		if err == nil {
			_ = action.SendAck()
		}
		return err
	}
	return nil
}
