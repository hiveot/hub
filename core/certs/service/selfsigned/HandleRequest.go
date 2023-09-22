package selfsigned

import (
	"github.com/hiveot/hub/api/go/certs"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"log/slog"
)

// HandleRequest handle incoming RPC requests for managing clients
func (svc *SelfSignedCertsService) HandleRequest(action *hubclient.RequestMessage) error {
	slog.Info("HandleRequest", slog.String("actionID", action.ActionID))

	// TODO: double-check the caller is an admin or svc
	switch action.ActionID {
	case certs.CreateDeviceCertAction:
		req := certs.CreateDeviceCertReq{}
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
	case certs.CreateServiceCertAction:
		req := certs.CreateServiceCertReq{}
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
	case certs.CreateUserCertAction:
		req := certs.CreateUserCertReq{}
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
	case certs.VerifyCertAction:
		req := certs.VerifyCertReq{}
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
