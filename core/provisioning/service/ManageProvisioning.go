package service

import (
	"context"
	"github.com/hiveot/hub/pkg/provisioning"

	"github.com/sirupsen/logrus"
)

// AddOOBSecrets adds one or more OOB Secrets for pre-approval and automatic provisioning
// OOBSecrets are kept in-memory until restart or they expire
func (svc *ProvisioningService) AddOOBSecrets(_ context.Context, secrets []provisioning.OOBSecret) error {
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
func (svc *ProvisioningService) ApproveRequest(_ context.Context, deviceID string) error {
	logrus.Infof("deviceID=%s", deviceID)
	svc.mux.Lock()
	defer svc.mux.Unlock()
	svc.oobSecrets[deviceID] = provisioning.OOBSecret{
		DeviceID:  deviceID,
		OobSecret: ApprovedSecret,
	}
	return nil
}

// GetApprovedRequests returns the list of approved provisioning requests
func (svc *ProvisioningService) GetApprovedRequests(_ context.Context) (
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
func (svc *ProvisioningService) GetPendingRequests(_ context.Context) (
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

// Release the provided capabilities
// nothing to do here as they are centralized by the service
func (svc *ProvisioningService) Release() {
}
