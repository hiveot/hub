package provisioning

import (
	"context"

	"github.com/hiveot/hub/api/go/hubapi"
)

// ServiceName is the name of the store for socket connection and logging
const ServiceName = hubapi.ProvisioningServiceName

// OOBSecret holds a device's Out Of Band secret for automated provisioning
// If the deviceID and MD5 hash of the secret match with the request it will be approved immediately
type OOBSecret struct {

	// The unique device ID or MAC address
	DeviceID string

	// The OOB secret of the device, "approved" to accept any secret
	OobSecret string
}

// ProvisionStatus holds the status of a provisioning request
type ProvisionStatus struct {

	// deviceID is the required unique ID of the device on the local network.
	DeviceID string

	// CA's certificate for future secure TLS connections, in PEM format.
	CaCertPEM string

	// The issued client certificate if approved, in PEM format
	ClientCertPEM string

	// the request is pending. Wait retrySec seconds before retrying
	Pending bool

	// ISO8601 timestamp when the request was received. Used to expire requests.
	RequestTime string

	// Optional delay for retrying the request in seconds in case status is pending
	RetrySec int
}

// IProvisioning defines a POGS based interface of the provisioning service
type IProvisioning interface {

	// CapManageProvisioning provides the capability to manage provisioning requests
	CapManageProvisioning(ctx context.Context, clientID string) (IManageProvisioning, error)

	// CapRequestProvisioning provides the capability to provision IoT devices
	CapRequestProvisioning(ctx context.Context, clientID string) (IRequestProvisioning, error)

	// CapRefreshProvisioning provides the capability for IoT devices to refresh
	CapRefreshProvisioning(ctx context.Context, clientID string) (IRefreshProvisioning, error)

	// Release the client capability
	Release()
}

// IManageProvisioning provides the capability to manage provisioning requests and OOB secrets
type IManageProvisioning interface {

	// AddOOBSecrets adds a list of OOB secrets
	AddOOBSecrets(ctx context.Context, secrets []OOBSecret) error

	// ApproveRequest approves a pending request for the given device ID
	ApproveRequest(ctx context.Context, deviceID string) error

	// GetApprovedRequests returns a list of approved requests
	GetApprovedRequests(ctx context.Context) ([]ProvisionStatus, error)

	// GetPendingRequests returns a list of pending requests
	GetPendingRequests(ctx context.Context) ([]ProvisionStatus, error)

	// Release the capability and its resources after use
	Release()
}

// IRequestProvisioning defines the capability to request or refresh a provisioning certificate
// Intended for use by IoT devices.
type IRequestProvisioning interface {

	// SubmitProvisioningRequest handles the provisioning request.
	// If the deviceID and (MD5 hash of) the secret match with previously uploaded secrets then the
	// request will be approved immediately.
	// This returns an error if the request is invalid
	SubmitProvisioningRequest(ctx context.Context,
		deviceID string, md5Secret string, pubKeyPEM string) (ProvisionStatus, error)

	// Release the capability and its resources after use
	Release()
}

// IRefreshProvisioning defines the capability to refresh an existing certificate
// This is only available to IoT devices with an existing valid certificate.
type IRefreshProvisioning interface {
	// RefreshDeviceCert refreshes a device certificate with a new expiry date
	// If the certificate is still valid this will succeed. If the certificate is expired
	// then the request must be approved by the administrator.
	//
	//  certPEM the current certificate in PEM format
	//
	// Returns the provisioning status containing the new certificate.
	RefreshDeviceCert(ctx context.Context, certPEM string) (ProvisionStatus, error)

	// Release the capability and its resources after use
	Release()
}
