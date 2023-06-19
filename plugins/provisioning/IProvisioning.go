package provisioning

// ServiceName is the name of the store for socket connection and logging
const ServiceName = "provisioning"

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

	// AddOOBSecrets adds a list of OOB secrets
	AddOOBSecrets(secrets []OOBSecret) error

	// ApproveRequest approves a pending request for the given device ID
	ApproveRequest(deviceID string) error

	// GetApprovedRequests returns a list of approved requests
	GetApprovedRequests() ([]ProvisionStatus, error)

	// GetPendingRequests returns a list of pending requests
	GetPendingRequests() ([]ProvisionStatus, error)

	// SubmitProvisioningRequest handles the provisioning request.
	// If the deviceID and (MD5 hash of) the secret match with previously uploaded secrets then the
	// request will be approved immediately.
	// This returns an error if the request is invalid
	SubmitProvisioningRequest(
		deviceID string, md5Secret string, pubKeyPEM string) (ProvisionStatus, error)

	// RefreshDeviceCert refreshes a device certificate with a new expiry date
	// If the certificate is still valid this will succeed. If the certificate is expired
	// then the request must be approved by the administrator.
	//
	//  certPEM the current certificate in PEM format
	//
	// Returns the provisioning status containing the new certificate.
	RefreshDeviceCert(certPEM string) (ProvisionStatus, error)

	// Release the capability and its resources after use
	Release()
}
