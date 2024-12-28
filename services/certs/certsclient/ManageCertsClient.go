package certsclient

import (
	"github.com/hiveot/hub/services/certs/certsapi"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot/td"
)

// CertsClient is a marshaller for cert service messages using a provided hub connection.
// This uses the default serializer to marshal and unmarshal messages.
type CertsClient struct {
	// dThingID digital twin service ID of the certificate management
	dThingID string
	// Connection to the hub
	hc transports.IConsumerConnection
}

//// helper for publishing a rpc request to the certs service
//func (cl *CertsClient) pubReq(action string, req interface{}, resp interface{}) error {
//	var msg []byte
//	if req != nil {
//		msg, _ = jsoniter.Marshal(req)
//	}
//
//	data, err := cl.hc.PubRPCRequest(
//		cl.serviceID, certs.ManageCertsCapability, action, msg)
//	if err != nil {
//		return err
//	}
//	if data.ErrorReply != nil {
//		return data.ErrorReply
//	}
//	err = cl.hc.ParseResponse(data.Payload, resp)
//	return err
//}

// CreateDeviceCert generates or renews IoT device certificate for access hub IoT gateway
func (cl *CertsClient) CreateDeviceCert(deviceID string, pubKeyPEM string, validityDays int) (
	certPEM string, caCertPEM string, err error) {

	req := certsapi.CreateDeviceCertArgs{
		DeviceID:     deviceID,
		PubKeyPEM:    pubKeyPEM,
		ValidityDays: validityDays,
	}
	resp := certsapi.CreateCertResp{}
	err = cl.hc.InvokeAction(cl.dThingID, certsapi.CreateDeviceCertMethod, req, &resp)
	return resp.CertPEM, resp.CaCertPEM, err
}

// CreateServiceCert generates or renews service certificates for access hub IoT gateway
func (cl *CertsClient) CreateServiceCert(
	serviceID string, pubKeyPEM string, names []string, validityDays int) (
	certPEM string, caCertPEM string, err error) {

	req := certsapi.CreateServiceCertArgs{
		ServiceID:    serviceID,
		PubKeyPEM:    pubKeyPEM,
		Names:        names,
		ValidityDays: validityDays,
	}
	resp := certsapi.CreateCertResp{}
	err = cl.hc.InvokeAction(cl.dThingID, certsapi.CreateServiceCertMethod, req, &resp)

	return resp.CertPEM, resp.CaCertPEM, err
}

// CreateUserCert generates or renews user certificates for access hiveot hub
func (cl *CertsClient) CreateUserCert(
	userID string, pubKeyPEM string, validityDays int) (
	certPEM string, caCertPEM string, err error) {

	req := certsapi.CreateUserCertArgs{
		UserID:       userID,
		PubKeyPEM:    pubKeyPEM,
		ValidityDays: validityDays,
	}
	resp := certsapi.CreateCertResp{}
	err = cl.hc.InvokeAction(cl.dThingID, certsapi.CreateUserCertMethod, req, &resp)
	return resp.CertPEM, resp.CaCertPEM, err
}

// VerifyCert verifies if the certificate is valid for the Hub
func (cl *CertsClient) VerifyCert(
	clientID string, certPEM string) (err error) {

	req := certsapi.VerifyCertArgs{
		ClientID: clientID,
		CertPEM:  certPEM,
	}
	err = cl.hc.InvokeAction(cl.dThingID, certsapi.VerifyCertMethod, req, nil)
	return err
}

// NewCertsClient returns a certs service client for managing certificates
//
//	hc is the hub client connection to use
func NewCertsClient(hc transports.IConsumerConnection) *CertsClient {
	agentID := certsapi.CertsAdminAgentID

	cl := CertsClient{
		hc:       hc,
		dThingID: td.MakeDigiTwinThingID(agentID, certsapi.CertsAdminThingID),
	}
	return &cl
}
