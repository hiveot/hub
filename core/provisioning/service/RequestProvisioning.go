package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/hiveot/hub/pkg/provisioning"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/hiveot/hub/lib/certsclient"
)

// SubmitProvisioningRequest handles provisioning request
// * If the request is not approved, it is added to the list of pending requests
// * If the request is approved, then it is removed from the list of pending requests
// * If the secret doesn't match the status is pending and the request will be added to the list of pending requests
func (svc *ProvisioningService) SubmitProvisioningRequest(ctx context.Context,
	deviceID string, secretMD5 string, pubKeyPEM string) (provStatus provisioning.ProvisionStatus, err error) {

	var approved = false
	provStatus.DeviceID = deviceID
	provStatus.RequestTime = time.Now().Format(time.RFC3339)
	provStatus.Pending = true
	provStatus.RetrySec = DefaultRetrySec
	//provStatus.CaCertPEM = svc.caCertPEM

	// validate parameters
	if deviceID == "" {
		err = fmt.Errorf("provisioning request failed. Missing deviceID")
		logrus.Warning(err)
		return provStatus, err
	}
	_, err = certsclient.PublicKeyFromPEM(pubKeyPEM)
	if err != nil {
		err = fmt.Errorf("invalid public key for device '%s': %s", deviceID, err)
		logrus.Warning(err)
		return provStatus, err
	}

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

	// if a request is approved, create a certificate
	approved, err = svc.verifyApproval(deviceID, secretMD5)
	if approved {
		clientCertPEM, caCertPEM, err2 := svc.certCapability.CreateDeviceCert(
			ctx, deviceID, pubKeyPEM, DefaultIoTCertValidityDays)
		provStatus.ClientCertPEM = clientCertPEM
		provStatus.CaCertPEM = caCertPEM
		provStatus.Pending = false
		provStatus.RetrySec = 0
		err = err2
		if err2 == nil {
			// remove the request from pending requests and add it to the approved
			svc.mux.Lock()
			delete(svc.pending, deviceID)
			svc.approved[deviceID] = provStatus
			svc.mux.Unlock()
			logrus.Infof("Request for device certificate is approved. DeviceID: '%s'", deviceID)
		}
	} else {
		// not yet approved or mismatched secret
		// so, add the request to the list of pending requests
		provStatus.Pending = true
		provStatus.RetrySec = DefaultRetrySec
		// this is not an error
		svc.mux.Lock()
		svc.pending[deviceID] = provStatus
		svc.mux.Unlock()
		if err != nil {
			// This is not an error, the request will be pending
			logrus.Warningf("Request for device certificate is pending, reason: %s", err)
			err = nil
		} else {
			logrus.Warningf("Request for device certificate is pending. DeviceID: '%s'", deviceID)
		}
	}
	return provStatus, err
}

// Verify if the provisioning request is approved
func (svc *ProvisioningService) verifyApproval(deviceID string, secretMD5 string) (approved bool, err error) {
	// check for manual approval
	secretToMatch, hasSecret := svc.oobSecrets[deviceID]
	if !hasSecret {
		// no known secret, so add the request for manual approval
		approved = false
	} else if secretToMatch.OobSecret == ApprovedSecret {
		// manual approval in place.
		approved = true

	} else {
		md5ToMatch := fmt.Sprint(md5.Sum([]byte(secretToMatch.OobSecret)))
		if secretMD5 == md5ToMatch {
			approved = true
		} else {
			// not a matching secret, reject the request
			approved = false
			err = fmt.Errorf("secret doesn't match for device %s", deviceID)
		}
	}
	return approved, err
}
