package certsclient

import (
	"github.com/hiveot/hub/api/go/certs"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
)

// CertsSvcClient is a marshaller for cert service messages using a provided hub connection.
// This uses the default serializer to marshal and unmarshal messages.
type CertsSvcClient struct {
	// ID of the certs service that handles the requests
	serviceID string
	hc        hubclient.IHubClient
}

// helper for publishing a rpc request to the certs service
func (cl *CertsSvcClient) pubReq(action string, req interface{}, resp interface{}) error {
	var msg []byte
	if req != nil {
		msg, _ = ser.Marshal(req)
	}

	data, err := cl.hc.PubServiceRPC(cl.serviceID, certs.CertsManageCertsCapability, action, msg)
	if err != nil {
		return err
	}
	if data.ErrorReply != nil {
		return data.ErrorReply
	}
	err = cl.hc.ParseResponse(data.Payload, resp)
	return err
}

// CreateDeviceCert generates or renews IoT device certificate for access hub IoT gateway
func (cl *CertsSvcClient) CreateDeviceCert(deviceID string, pubKeyPEM string, validityDays int) (
	certPEM string, caCertPEM string, err error) {

	req := certs.CreateDeviceCertReq{
		DeviceID:     deviceID,
		PubKeyPEM:    pubKeyPEM,
		ValidityDays: validityDays,
	}
	resp := certs.CreateCertResp{}
	err = cl.pubReq(certs.CreateDeviceCertAction, req, &resp)
	return resp.CertPEM, resp.CaCertPEM, err
}

// CreateServiceCert generates or renews service certificates for access hub IoT gateway
func (cl *CertsSvcClient) CreateServiceCert(
	serviceID string, pubKeyPEM string, names []string, validityDays int) (
	certPEM string, caCertPEM string, err error) {

	req := certs.CreateServiceCertReq{
		ServiceID:    serviceID,
		PubKeyPEM:    pubKeyPEM,
		Names:        names,
		ValidityDays: validityDays,
	}
	resp := certs.CreateCertResp{}
	err = cl.pubReq(certs.CreateServiceCertAction, req, &resp)
	return resp.CertPEM, resp.CaCertPEM, err
}

// CreateUserCert generates or renews user certificates for access hiveot hub
func (cl *CertsSvcClient) CreateUserCert(
	userID string, pubKeyPEM string, validityDays int) (
	certPEM string, caCertPEM string, err error) {

	req := certs.CreateUserCertReq{
		UserID:       userID,
		PubKeyPEM:    pubKeyPEM,
		ValidityDays: validityDays,
	}
	resp := certs.CreateCertResp{}
	err = cl.pubReq(certs.CreateUserCertAction, req, &resp)
	return resp.CertPEM, resp.CaCertPEM, err
}

// VerifyCert verifies if the certificate is valid for the Hub
func (cl *CertsSvcClient) VerifyCert(
	clientID string, certPEM string) (err error) {

	req := certs.VerifyCertReq{
		ClientID: clientID,
		CertPEM:  certPEM,
	}
	err = cl.pubReq(certs.VerifyCertAction, req, nil)
	return err
}

// NewCertsSvcClient returns an certs service client for managing certificates
//
//	hc is the hub client connection to use
func NewCertsSvcClient(hc hubclient.IHubClient) *CertsSvcClient {
	serviceID := certs.ServiceName
	cl := CertsSvcClient{
		hc:        hc,
		serviceID: serviceID,
	}
	return &cl
}
