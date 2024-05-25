package idprovapi

import "github.com/hiveot/hub/runtime/api"

// AgentID is the connect ID of the agent connecting to the Hub
const AgentID = "idprov"

// ManageServiceID is the ID of the service as used by the agent
const ManageServiceID = "manage"

// Manage methods
const (
	// ApproveRequestMethod approves a provisioning request.
	// This does not add the device until a provisioning request is made
	ApproveRequestMethod = "approveRequest"

	// GetRequestsMethod returns a list of provisioning requests
	// This is an in-memory list that is cleared when the service restarts
	GetRequestsMethod = "getRequests"

	// PreApproveClientsMethod uploads a list of pre-approved devices or services
	// This list remains active until the service restarts.
	// Devices are not added until the request is received and accepted.
	PreApproveClientsMethod = "preApproveClients"

	// RejectRequestMethod rejects a provisioning request
	RejectRequestMethod = "rejectRequest"

	// SubmitRequestMethod submits a provisioning request
	SubmitRequestMethod = "submitRequest"
)

type ApproveRequestArgs struct {
	// ClientID of an approved device or service
	ClientID string `json:"clientID"`
	// ClientType to assign to the approval
	ClientType api.ClientType `json:"clientType"`
}

type GetRequestsArgs struct {
	// include accepted,pending and/or rejected requests
	Approved bool `json:"approved,omitempty"`
	Pending  bool `json:"pending,omitempty"`
	Rejected bool `json:"rejected,omitempty"`
}

type GetRequestsResp struct {
	Requests []ProvisionStatus `json:"requests"`
}

type PreApprovedClient struct {
	// AgentID of a pre-approved device or service
	ClientID string `json:"clientID,omitempty"`
	// client is a device or service
	ClientType api.ClientType `json:"clientType"`
	// Optional MAC for extra checking
	MAC string `json:"mac"`
	// Device or service public key used to issue tokens
	PubKey string `json:"pubKey"`
}
type PreApproveClientsArgs struct {
	Approvals []PreApprovedClient `json:"approvals,omitempty"`
}

type RejectRequestArgs struct {
	// ClientID of an rejected device or service
	ClientID string `json:"clientID,omitempty"`
}

type SubmitRequestArgs struct {
	// ClientID of the device or service
	ClientID string `json:"clientID,omitempty"`
	// client is a device or service (default is ClientTypeDevice)
	ClientType api.ClientType `json:"clientType"`
	// Optional MAC if available
	MAC string `json:"mac"`
	// Device or service public key used to issue tokens (required)
	PubKey string `json:"pubKey"`
}
