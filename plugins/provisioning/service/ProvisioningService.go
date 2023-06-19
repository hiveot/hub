package service

import (
	"crypto/md5"
	"fmt"
	"github.com/hiveot/hub/core/certs"
	"github.com/hiveot/hub/lib/certsclient"
	"github.com/hiveot/hub/plugins/provisioning"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slog"
	"sync"
	"time"
)

const DefaultIoTCertValidityDays = 14
const ApprovedSecret = "approved"
const DefaultRetrySec = 12 * 3600

// ProvisioningService handles the provisioning requests.
// This implements the IProvisioning interface.
//
// This verifies requests against the out-of-bound secret and uses the certificate service to
// issue IoT device certificates.
// If no OOB secret is provided, the request is stored and awaits approval by the administrator.
//
// If enabled, a discovery record is published using DNS-SD to allow potential clients to find the
// address and ports of the provisioning server, and optionally additional services.
type ProvisioningService struct {
	// client to certificate service
	certsSvc certs.ICerts

	// runtime status
	oobSecrets map[string]provisioning.OOBSecret // [deviceID]secret simple in-memory store for OOB secrets
	pending    map[string]provisioning.ProvisionStatus
	approved   map[string]provisioning.ProvisionStatus

	// mutex to guard access to maps
	mux sync.RWMutex
}

// AddOOBSecrets adds one or more OOB Secrets for pre-approval and automatic provisioning
// OOBSecrets are kept in-memory until restart or they expire
func (svc *ProvisioningService) AddOOBSecrets(secrets []provisioning.OOBSecret) error {
	logrus.Infof("count=%d", len(secrets))

	svc.mux.Lock()
	defer svc.mux.Unlock()
	for _, secret := range secrets {
		svc.oobSecrets[secret.DeviceID] = secret
	}
	return nil
}

// ApproveRequest approves a pending request
// The next time the request is made, it will be accepted
func (svc *ProvisioningService) ApproveRequest(deviceID string) error {
	logrus.Infof("deviceID=%s", deviceID)
	svc.mux.Lock()
	defer svc.mux.Unlock()
	svc.oobSecrets[deviceID] = provisioning.OOBSecret{
		DeviceID:  deviceID,
		OobSecret: ApprovedSecret,
	}
	return nil
}

// RefreshDeviceCert renews the given device certificate
// The given certificate must be valid, otherwise a new provisioning request should be made.
func (svc *ProvisioningService) RefreshDeviceCert(certPEM string) (
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

		err = svc.certsSvc.VerifyCert(deviceID, certPEM)
	}
	if err == nil {
		// create a new certificate
		pubKeyPEM, err = certsclient.PublicKeyToPEM(cert.PublicKey)
	}
	if err == nil {
		provStatus.DeviceID = deviceID
		provStatus.ClientCertPEM, provStatus.CaCertPEM, err =
			svc.certsSvc.CreateDeviceCert(deviceID, pubKeyPEM, DefaultIoTCertValidityDays)
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
		slog.Info("success", "deviceID", deviceID)
	}
	return provStatus, err
}

// GetApprovedRequests returns the list of approved provisioning requests
func (svc *ProvisioningService) GetApprovedRequests() (
	[]provisioning.ProvisionStatus, error) {

	result := make([]provisioning.ProvisionStatus, 0)
	svc.mux.RLock()
	defer svc.mux.RUnlock()
	for _, req := range svc.approved {
		result = append(result, req)
	}
	logrus.Infof("count=%d", len(result))
	return result, nil
}

// GetPendingRequests returns the list of open requests
func (svc *ProvisioningService) GetPendingRequests() (
	[]provisioning.ProvisionStatus, error) {

	result := make([]provisioning.ProvisionStatus, 0)
	svc.mux.RLock()
	defer svc.mux.RUnlock()
	for _, req := range svc.pending {
		result = append(result, req)
	}
	logrus.Infof("count=%d", len(result))
	return result, nil
}

// SubmitProvisioningRequest handles provisioning request
// * If the request is not approved, it is added to the list of pending requests
// * If the request is approved, then it is removed from the list of pending requests
// * If the secret doesn't match the status is pending and the request will be added to the list of pending requests
func (svc *ProvisioningService) SubmitProvisioningRequest(
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
		slog.Warn("invalid key", "err", err)
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
		clientCertPEM, caCertPEM, err2 := svc.certsSvc.CreateDeviceCert(
			deviceID, pubKeyPEM, DefaultIoTCertValidityDays)
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

func (svc *ProvisioningService) Start() error {
	return nil
}

func (svc *ProvisioningService) Stop() error {
	logrus.Infof("Stopping Provisioning service")
	return nil
}

// NewProvisioningService creates a new provisioning service instance
// This requires the capability to obtain and verify device certificates
// Invoke 'Stop' when done to close the provided certCap and verifyCap capabilities
func NewProvisioningService(certsSvc certs.ICerts) *ProvisioningService {
	svc := &ProvisioningService{
		certsSvc:   certsSvc,
		oobSecrets: make(map[string]provisioning.OOBSecret),
		pending:    make(map[string]provisioning.ProvisionStatus),
		approved:   make(map[string]provisioning.ProvisionStatus),
	}

	return svc
}
