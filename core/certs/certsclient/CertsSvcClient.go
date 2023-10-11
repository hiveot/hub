package certsclient

import (
	"github.com/hiveot/hub/core/certs"
	"github.com/hiveot/hub/lib/hubclient"
)

// CertsSvcClient is a marshaller for cert service messages using a provided hub connection.
// This uses the default serializer to marshal and unmarshal messages.
type CertsSvcClient struct {
	// agent handling the request
	agentID string
	// directory capability to use
	capID string
	hc    hubclient.IHubClient
}

//// helper for publishing a rpc request to the certs service
//func (cl *CertsSvcClient) pubReq(action string, req interface{}, resp interface{}) error {
//	var msg []byte
//	if req != nil {
//		msg, _ = ser.Marshal(req)
//	}
//
//	data, err := cl.hc.PubRPCRequest(
//		cl.serviceID, certs.CertsManageCertsCapability, action, msg)
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
func (cl *CertsSvcClient) CreateDeviceCert(deviceID string, pubKeyPEM string, validityDays int) (
	certPEM string, caCertPEM string, err error) {

	req := certs.CreateDeviceCertArgs{
		DeviceID:     deviceID,
		PubKeyPEM:    pubKeyPEM,
		ValidityDays: validityDays,
	}
	resp := certs.CreateCertResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, certs.CreateDeviceCertReq, req, &resp)
	return resp.CertPEM, resp.CaCertPEM, err
}

// CreateServiceCert generates or renews service certificates for access hub IoT gateway
func (cl *CertsSvcClient) CreateServiceCert(
	serviceID string, pubKeyPEM string, names []string, validityDays int) (
	certPEM string, caCertPEM string, err error) {

	req := certs.CreateServiceCertArgs{
		ServiceID:    serviceID,
		PubKeyPEM:    pubKeyPEM,
		Names:        names,
		ValidityDays: validityDays,
	}
	resp := certs.CreateCertResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, certs.CreateServiceCertReq, req, &resp)

	return resp.CertPEM, resp.CaCertPEM, err
}

// CreateUserCert generates or renews user certificates for access hiveot hub
func (cl *CertsSvcClient) CreateUserCert(
	userID string, pubKeyPEM string, validityDays int) (
	certPEM string, caCertPEM string, err error) {

	req := certs.CreateUserCertArgs{
		UserID:       userID,
		PubKeyPEM:    pubKeyPEM,
		ValidityDays: validityDays,
	}
	resp := certs.CreateCertResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, certs.CreateUserCertReq, req, &resp)
	return resp.CertPEM, resp.CaCertPEM, err
}

// VerifyCert verifies if the certificate is valid for the Hub
func (cl *CertsSvcClient) VerifyCert(
	clientID string, certPEM string) (err error) {

	req := certs.VerifyCertArgs{
		ClientID: clientID,
		CertPEM:  certPEM,
	}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, certs.VerifyCertReq, req, nil)
	return err
}

// NewCertsSvcClient returns an certs service client for managing certificates
//
//	hc is the hub client connection to use
func NewCertsSvcClient(hc hubclient.IHubClient) *CertsSvcClient {
	cl := CertsSvcClient{
		hc:      hc,
		agentID: certs.ServiceName,
		capID:   certs.CertsManageCertsCapability,
	}
	return &cl
}
