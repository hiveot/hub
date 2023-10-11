package service

import (
	"context"
	"fmt"
	"github.com/hiveot/hub/pkg/provisioning"
	"time"

	"github.com/hiveot/hub/lib/certsclient"
	"github.com/sirupsen/logrus"
)

// RefreshDeviceCert renews the given device certificate
// The given certificate must be valid, otherwise a new provisioning request should be made.
func (svc *ProvisioningService) RefreshDeviceCert(ctx context.Context, certPEM string) (
	provStatus provisioning.ProvisionStatus, err error) {

	var pubKeyPEM string
	var deviceID string

	// If the client connect with a valid certificate then no need to verify the signature using a secret
	// a Thing OU certificate must be valid and have the same deviceID
	// a Plugin or Admin OU certificate can issue a certificate without an existing deviceID
	//for _, peerCert = range req.TLS.PeerCertificates {
	//	err = srv.validateCertificate(peerCert, provReq.DeviceID)
	//	if err == nil {
	//		validCert = true
	//		break
	//	}
	//}
	cert, err := certsclient.X509CertFromPEM(certPEM)
	if err == nil {

		// TODO: The deviceID of the certificate must match the caller
		deviceID = cert.Subject.CommonName
		callerID := ctx.Value("callerID")
		_ = callerID

		err = svc.verifyCapability.VerifyCert(ctx, deviceID, certPEM)
	}
	if err == nil {
		// create a new certificate
		pubKeyPEM, err = certsclient.PublicKeyToPEM(cert.PublicKey)
	}
	if err == nil {
		provStatus.DeviceID = deviceID
		provStatus.ClientCertPEM, provStatus.CaCertPEM, err =
			svc.certCapability.CreateDeviceCert(ctx, deviceID, pubKeyPEM, DefaultIoTCertValidityDays)
		provStatus.Pending = false
		provStatus.RetrySec = 0
		provStatus.RequestTime = time.Now().Format(time.RFC3339)
		// add to approved requests
		svc.mux.Lock()
		svc.approved[deviceID] = provStatus
		svc.mux.Unlock()

	}
	if err != nil {
		err = fmt.Errorf("deviceID=%s Failed: %s", deviceID, err)
	} else {
		logrus.Infof("deviceID=%s. Success.", deviceID)
	}
	return provStatus, err
}
