package service

import (
	"context"
	"github.com/hiveot/hub/pkg/certs"
	"github.com/hiveot/hub/pkg/provisioning"
	"sync"

	"github.com/sirupsen/logrus"
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
	// capability to create device certificates
	certCapability certs.IDeviceCerts // client with capability to create device certificates
	// capability to verify device certificates
	verifyCapability certs.IVerifyCerts

	// runtime status
	oobSecrets map[string]provisioning.OOBSecret // [deviceID]secret simple in-memory store for OOB secrets
	pending    map[string]provisioning.ProvisionStatus
	approved   map[string]provisioning.ProvisionStatus

	// mutex to guard access to maps
	mux sync.RWMutex
}

// CapManageProvisioning provides the capability to manage provisioning
func (svc *ProvisioningService) CapManageProvisioning(
	_ context.Context, clientID string) (provisioning.IManageProvisioning, error) {
	logrus.Infof("clientID=%s", clientID)
	// TODO: separate instances of each capability
	return svc, nil
}

// CapRefreshProvisioning provides the capability to refresh device provisioning
func (svc *ProvisioningService) CapRefreshProvisioning(
	_ context.Context, clientID string) (provisioning.IRefreshProvisioning, error) {
	logrus.Infof("clientID=%s", clientID)
	// TODO: separate instances of each capability
	return svc, nil
}

// CapRequestProvisioning provides the capability to request device provisioning
func (svc *ProvisioningService) CapRequestProvisioning(
	_ context.Context, clientID string) (provisioning.IRequestProvisioning, error) {
	logrus.Infof("clientID=%s", clientID)
	// TODO: separate instances of each capability and lifecycle
	return svc, nil
}

func (svc *ProvisioningService) Start(_ context.Context) error {
	return nil
}

func (svc *ProvisioningService) Stop() error {
	logrus.Infof("Stopping Provisioning service")
	svc.certCapability.Release()
	svc.verifyCapability.Release()
	return nil
}

// NewProvisioningService creates a new provisioning service instance
// This requires the capability to obtain and verify device certificates
// Invoke 'Stop' when done to close the provided certCap and verifyCap capabilities
func NewProvisioningService(certCap certs.IDeviceCerts, verifyCap certs.IVerifyCerts) *ProvisioningService {
	svc := &ProvisioningService{
		certCapability:   certCap,
		verifyCapability: verifyCap,
		oobSecrets:       make(map[string]provisioning.OOBSecret),
		pending:          make(map[string]provisioning.ProvisionStatus),
		approved:         make(map[string]provisioning.ProvisionStatus),
	}

	return svc
}
