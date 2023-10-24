package certsclient

import (
	"github.com/hiveot/hub/core/certs/certsapi"
	"github.com/hiveot/hub/lib/hubclient"
)

// CertsClient is a marshaller for cert service messages using a provided hub connection.
// This uses the default serializer to marshal and unmarshal messages.
type CertsClient struct {
	// agent handling the request
	agentID string
	// directory capability to use
	capID string
	hc    hubclient.IHubClient
}

//// helper for publishing a rpc request to the certs service
//func (cl *CertsClient) pubReq(action string, req interface{}, resp interface{}) error {
//	var msg []byte
//	if req != nil {
//		msg, _ = ser.Marshal(req)
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
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, certsapi.CreateDeviceCertMethod, req, &resp)
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
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, certsapi.CreateServiceCertMethod, req, &resp)

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
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, certsapi.CreateUserCertMethod, req, &resp)
	return resp.CertPEM, resp.CaCertPEM, err
}

// VerifyCert verifies if the certificate is valid for the Hub
func (cl *CertsClient) VerifyCert(
	clientID string, certPEM string) (err error) {

	req := certsapi.VerifyCertArgs{
		ClientID: clientID,
		CertPEM:  certPEM,
	}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, certsapi.VerifyCertMethod, req, nil)
	return err
}

// NewCertsClient returns a certs service client for managing certificates
//
//	hc is the hub client connection to use
func NewCertsClient(hc hubclient.IHubClient) *CertsClient {
	cl := CertsClient{
		hc:      hc,
		agentID: certsapi.ServiceName,
		capID:   certsapi.ManageCertsCapability,
	}
	return &cl
}
