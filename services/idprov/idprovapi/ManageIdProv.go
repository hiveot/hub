package idprovapi

import "github.com/hiveot/hub/runtime/api"

// AgentID is the name of the agent connecting to the Hub
const AgentID = "idprov"

// ManageProvisioningThingID is the service ThingID that manages provisioning via the Hub
const ManageProvisioningThingID = "manageIdProv"

// ApproveRequestMethod approves a provisioning request.
// This does not add the device until a provisioning request is made
const ApproveRequestMethod = "approveRequest"

type ApproveRequestArgs struct {
	// ClientID of an approved device or service
	ClientID string `json:"clientID"`
	// ClientType to assign to the approval
	ClientType string `json:"clientType"`
}

// GetRequestsMethod returns a list of provisioning requests
// This is an in-memory list that is cleared when the service restarts
const GetRequestsMethod = "getRequests"

type GetRequestsArgs struct {
	// include accepted,pending and/or rejected requests
	Approved bool `json:"approved,omitempty"`
	Pending  bool `json:"pending,omitempty"`
	Rejected bool `json:"rejected,omitempty"`
}

type GetRequestsResp struct {
	Requests []ProvisionStatus `json:"requests"`
}

// PreApproveClientsMethod uploads a list of pre-approved devices or services
// This list remains active until the service restarts.
// Devices are not added until the request is received and accepted.
const PreApproveClientsMethod = "preApproveClients"

type PreApprovedClient struct {
	// ClientID of a pre-approved device or service
	ClientID string `json:"clientID,omitempty"`
	// client is a device or service
	ClientType string `json:"clientType"`
	// Optional MAC for extra checking
	MAC string `json:"mac"`
	// Device or service public key used to issue tokens
	PubKey string `json:"pubKey"`
}
type PreApproveClientsArgs struct {
	Approvals []PreApprovedClient `json:"approvals,omitempty"`
}

// RejectRequestMethod rejects a provisioning request
const RejectRequestMethod = "rejectRequest"

type RejectRequestArgs struct {
	// ClientID of an rejected device or service
	ClientID string `json:"clientID,omitempty"`
}

// SubmitRequestMethod submits a provisioning request
const SubmitRequestMethod = "submitRequest"

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
