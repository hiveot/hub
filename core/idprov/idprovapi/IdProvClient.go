package idprovapi

// ProvisioningServerType defines the discovery type for the provisioning
// this will be published as _provisioning._hiveot._tcp
const ProvisioningServerType = "idprov"

// ProvisionStatus holds the status of a provisioning request
type ProvisionStatus struct {
	// ClientID is the unique ID of the device or service on the local network.
	ClientID string `json:"clientID,omitempty"`

	// ClientType is either ClientTypeDevice or ClientTypeService
	ClientType string `json:"clientType"`

	// PubKey holds the device's public key
	PubKey string `json:"pubKey"`

	// MAC contains the device's MAC address either for pre-approval or when issuing a token
	MAC string `json:"mac"`

	// the request is pending. Wait retrySec seconds before retrying
	Pending bool `json:"pending,omitempty"`

	// timestamp in msec since epoc when the request was approved.
	ApprovedMSE int64 `json:"approvedMSE,omitempty"`

	// timestamp in msec since epoc when the request was received. Used to expire requests.
	ReceivedMSE int64 `json:"receivedMSE,omitempty"`

	// timestamp in msec since epoc when the request was rejected.
	RejectedMSE int64 `json:"rejectedMSE,omitempty"`

	// Optional delay for retrying the request in seconds in case status is pending
	RetrySec int `json:"retrySec,omitempty"`
}

// ProvisionRequestPath to request provisioning through the HTTP endpoint
const ProvisionRequestPath = "/idprov/request"

// ProvisionRequestArgs arguments to request provisioning
type ProvisionRequestArgs struct {
	// ClientID of the device or service under which it will connect and publish events
	ClientID string `json:"clientID,omitempty"`
	// MAC address of the device
	MAC string `json:"mac"`
	// The device or service public key
	PubKey string `json:"pubKey,omitempty"`
}

// ProvisionRequestResp holds the response to the request
type ProvisionRequestResp struct {
	// The current status
	Status ProvisionStatus `json:"status"`
	// Token when approved.
	// This has a short lifespan and must be refresh immediately after connecting to the hub.
	Token string `json:"token,omitempty"`
}
