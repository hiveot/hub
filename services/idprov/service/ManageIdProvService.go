package service

import (
	"fmt"
	"github.com/hiveot/hub/messaging"
	authn "github.com/hiveot/hub/runtime/authn/api"
	"github.com/hiveot/hub/services/idprov/idprovapi"
	"log/slog"
	"sync"
	"time"
)

type ManageIdProvService struct {

	// request status by deviceID
	// [deviceID] pub-key simple in-memory store
	requests map[string]idprovapi.ProvisionStatus

	//
	ag *messaging.Agent
	// mutex to guard access to maps
	mux sync.RWMutex
}

// ApproveRequest approves an existing provisioning request.
// The client will be added on the next request.
// The next repeat request will return a short-lived token.
func (svc *ManageIdProvService) ApproveRequest(
	senderID string, args *idprovapi.ApproveRequestArgs) error {
	svc.mux.Lock()
	defer svc.mux.Unlock()

	slog.Info("ApproveRequest",
		slog.String("senderID", senderID),
		slog.String("deviceID", args.ClientID))
	status, found := svc.requests[args.ClientID]
	if !found {
		return fmt.Errorf("provisioning request for device '%s' not found", args.ClientID)
	}
	status.Pending = false
	status.ClientType = args.ClientType
	status.ApprovedMSE = time.Now().UnixMilli()
	status.RejectedMSE = 0
	svc.requests[args.ClientID] = status
	return nil
}

// GetRequests returns list of requests since last start
// If args.OnlyPending is set then only return pending requests
// Note that rejected requests are never returned
func (svc *ManageIdProvService) GetRequests(
	senderID string, args *idprovapi.GetRequestsArgs) (*idprovapi.GetRequestsResp, error) {
	svc.mux.RLock()
	defer svc.mux.RUnlock()

	resp := &idprovapi.GetRequestsResp{
		Requests: make([]idprovapi.ProvisionStatus, 0, len(svc.requests)),
	}
	for _, status := range svc.requests {
		if (status.Pending && args.Pending) ||
			(status.ApprovedMSE != 0 && args.Approved) ||
			(status.RejectedMSE != 0 && args.Rejected) {
			resp.Requests = append(resp.Requests, status)
		}
	}
	return resp, nil
}

// PreApproveClients uploads list of pre-approved devices and services
func (svc *ManageIdProvService) PreApproveClients(
	senderID string, args *idprovapi.PreApproveClientsArgs) error {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	slog.Info("PreApproveClients",
		slog.String("senderID", senderID),
		slog.Int("count", len(args.Approvals)))

	for _, approval := range args.Approvals {
		if approval.ClientID == "" {
			slog.Warn("PreApproval of client without clientID", "clientID", senderID)
		} else {
			svc.requests[approval.ClientID] = idprovapi.ProvisionStatus{
				ClientID:    approval.ClientID,
				ClientType:  approval.ClientType,
				PubKey:      approval.PubKey,
				MAC:         approval.MAC,
				Pending:     false,
				ApprovedMSE: time.Now().UnixMilli(),
			}
		}
	}
	return nil
}

// RejectRequest rejects a provisioning request
func (svc *ManageIdProvService) RejectRequest(
	senderID string, args *idprovapi.RejectRequestArgs) error {
	svc.mux.Lock()
	defer svc.mux.Unlock()

	slog.Info("RejectRequest",
		slog.String("senderID", senderID),
		slog.String("deviceID", args.ClientID))
	status, found := svc.requests[args.ClientID]
	if !found {
		return fmt.Errorf("provisioning request for client '%s' not found", args.ClientID)
	}
	status.Pending = false
	status.RejectedMSE = time.Now().UnixMilli()
	svc.requests[args.ClientID] = status
	return nil
}

// SubmitRequest creates a provisioning request for a device
//
// If the request is pre-approved a token will be returned if the pubKey and/or
// MAC matches.
// If the pre-approval does not include a public key then only match required is the MAC.
func (svc *ManageIdProvService) SubmitRequest(senderID string, args *idprovapi.ProvisionRequestArgs) (
	resp *idprovapi.ProvisionRequestResp, err error) {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	var token string

	slog.Info("SubmitRequest",
		slog.String("senderID", senderID),
		slog.String("deviceID", args.ClientID))
	status, found := svc.requests[args.ClientID]
	if !found {
		// new request
		status = idprovapi.ProvisionStatus{
			ClientID:    args.ClientID,
			PubKey:      args.PubKey,
			MAC:         args.MAC,
			Pending:     true,
			ReceivedMSE: time.Now().UnixMilli(),
			RetrySec:    60,
		}
	} else if status.ApprovedMSE != 0 {
		// (pre)approved request, add the user and issue a token
		status.ReceivedMSE = time.Now().UnixMilli()
		// public key or mac must match if provided
		if status.PubKey != "" && status.PubKey != args.PubKey {
			err = fmt.Errorf(
				"approval for '%s' denied as public key doesn't match", args.ClientID)
		} else if status.MAC != "" && status.MAC != args.MAC {
			err = fmt.Errorf(
				"approval for '%s' denied as mac address doesn't match", args.ClientID)
		} else if args.PubKey == "" {
			err = fmt.Errorf(
				"approval for '%s' denied as no public key was provided", args.ClientID)
		} else {
			err = nil
		}
		if err != nil {
			slog.Warn(err.Error(), "clientID", args.ClientID)
			return nil, err
		}

		status.Pending = false
		token, err = authn.AdminAddAgent(svc.ag.Consumer, status.ClientID, status.ClientID, status.PubKey)

		if err != nil {
			return nil, err
		}
		slog.Warn("provisioning token created for client", "clientID", args.ClientID)

	} else if status.RejectedMSE != 0 {
		// rejected request, ignore request
		// delay next request to 1 hour
		status.RetrySec = 3600
		status.ReceivedMSE = time.Now().UnixMilli()
		status.Pending = false
	} else {
		// repeat of request, update the request received timestamp and increase retry
		status.Pending = true
		status.ReceivedMSE = time.Now().UnixMilli()
		// delay next request to a maximum of 10 minutes
		if status.RetrySec < 600 {
			status.RetrySec += 30
		}
	}
	svc.requests[args.ClientID] = status
	resp = &idprovapi.ProvisionRequestResp{
		Status: status,
		Token:  token,
	}
	return resp, nil
}
func (svc *ManageIdProvService) Stop() {
}

func StartManageIdProvService(ag *messaging.Agent) (*ManageIdProvService, error) {

	svc := &ManageIdProvService{
		// map of requests by SenderID
		requests: make(map[string]idprovapi.ProvisionStatus),
		ag:       ag,
	}

	return svc, nil
}
