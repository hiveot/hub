package authservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
)

// AuthManageClients handles management of devices,users and service clients
// This implements the IAuthManageClients interface.
type AuthManageClients struct {
	// clients storage
	store auth.IAuthnStore
	// message server to apply changes to
	msgServer msgserver.IMsgServer
	// messaging client for receiving requests
	hc hubclient.IHubClient
	// subscription to receive requests
	mngSub hubclient.ISubscription
}

// AddDevice adds an IoT device and generates an authentication token
// This is handled by the underlying messaging core.
func (svc *AuthManageClients) AddDevice(
	deviceID string, name string, pubKey string) (token string, err error) {
	slog.Info("AddDevice",
		slog.String("deviceID", deviceID),
		slog.String("name", name),
		slog.String("pubKey", pubKey))

	if deviceID == "" {
		return "", fmt.Errorf("AddDevice: missing device ID")
	}
	// store/update device.
	err = svc.store.Add(deviceID, auth.ClientProfile{
		ClientID:    deviceID,
		ClientType:  auth.ClientTypeDevice,
		DisplayName: name,
		PubKey:      pubKey,
		Role:        auth.ClientRoleNone,
	})
	if err != nil {
		return "", err
	}
	// the token will be applied when authorization (group membership) is set
	svc.onChange()
	return pubKey, err
}

// AddService adds or updates a service with the admin role
func (svc *AuthManageClients) AddService(
	serviceID string, name string, pubKey string) (token string, err error) {
	slog.Info("AddService",
		slog.String("serviceID", serviceID),
		slog.String("name", name),
		slog.String("pubKey", pubKey))

	if serviceID == "" {
		return "", fmt.Errorf("missing service ID")
	}
	err = svc.store.Add(serviceID, auth.ClientProfile{
		ClientID:    serviceID,
		ClientType:  auth.ClientTypeService,
		DisplayName: name,
		PubKey:      pubKey,
		Role:        auth.ClientRoleAdmin,
	})
	if err != nil {
		return "", err
	}
	// the token will be applied when authorization (group membership) is set
	token = pubKey
	svc.onChange()
	return token, err
}

// AddUser adds a new user for password authentication
// If a public key is provided a signed token will be returned
func (svc *AuthManageClients) AddUser(
	userID string, userName string, password string, pubKey string, role string) (token string, err error) {

	slog.Info("AddUser",
		slog.String("userID", userID),
		slog.String("userName", userName),
		slog.String("pubKey", pubKey),
		slog.String("role", role))

	if userID == "" {
		return "", fmt.Errorf("missing user ID")
	}
	err = svc.store.Add(userID, auth.ClientProfile{
		ClientID:    userID,
		ClientType:  auth.ClientTypeUser,
		DisplayName: userName,
		PubKey:      pubKey,
		Role:        role,
	})
	if err != nil {
		return "", err
	}
	if password != "" {
		err = svc.store.SetPassword(userID, password)
		if err != nil {
			err = fmt.Errorf("AddUser: user '%s' added, but: %w. Continuing", userID, err)
			slog.Error(err.Error())
		}
	}
	// the token will be applied when authorization (group membership) is set
	token = pubKey
	svc.onChange()
	return token, err
}

func (svc *AuthManageClients) GetCount() (int, error) {
	return svc.store.Count(), nil
}

func (svc *AuthManageClients) GetAuthClientList() []msgserver.ClientAuthInfo {
	return svc.store.GetAuthClientList()
}

// GetProfile returns a client's profile
func (svc *AuthManageClients) GetProfile(clientID string) (profile auth.ClientProfile, err error) {
	entry, err := svc.store.GetProfile(clientID)
	return entry, err
}

// GetProfiles provide a list of known clients and their info.
func (svc *AuthManageClients) GetProfiles() (profiles []auth.ClientProfile, err error) {
	profiles, err = svc.store.GetProfiles()
	return profiles, err
}

// GetEntries provide a list of known clients and their info including bcrypted passwords
func (svc *AuthManageClients) GetEntries() (entries []auth.AuthnEntry) {
	return svc.store.GetEntries()
}

// handle authn management requests published by a hub manager
func (svc *AuthManageClients) HandleActions(action *hubclient.ActionMessage) error {
	slog.Info("handleActions",
		slog.String("actionID", action.ActionID))

	// TODO: doublecheck the caller is an admin or svc
	switch action.ActionID {
	case auth.AddDeviceAction:
		req := auth.AddDeviceReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		token, err := svc.AddDevice(req.DeviceID, req.DisplayName, req.PubKey)
		if err == nil {
			resp := auth.AddDeviceResp{Token: token}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case auth.AddServiceAction:
		req := auth.AddServiceReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		token, err := svc.AddService(req.ServiceID, req.DisplayName, req.PubKey)
		if err == nil {
			resp := auth.AddServiceResp{Token: token}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case auth.AddUserAction:
		req := auth.AddUserReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		token, err := svc.AddUser(
			req.UserID, req.DisplayName, req.Password, req.PubKey, req.Role)
		if err == nil {
			resp := auth.AddUserResp{Token: token}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case auth.GetCountAction:
		n, err := svc.GetCount()
		resp := auth.GetCountResp{N: n}
		reply, _ := ser.Marshal(&resp)
		action.SendReply(reply)
		return err
	case auth.GetProfileAction:
		req := auth.GetProfileReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		profile, err := svc.GetProfile(req.ClientID)
		if err == nil {
			resp := auth.GetProfileResp{Profile: profile}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case auth.GetProfilesAction:
		clientList, err := svc.GetProfiles()
		if err == nil {
			resp := auth.GetProfilesResp{Profiles: clientList}
			reply, _ := ser.Marshal(resp)
			action.SendReply(reply)
		}
		return err
	case auth.RemoveClientAction:
		req := &auth.RemoveClientReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.RemoveClient(req.ClientID)
		if err == nil {
			action.SendAck()
		}
		return err
	case auth.UpdateClientAction:
		req := &auth.UpdateClientReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.UpdateClient(req.ClientID, req.Profile)
		if err == nil {
			action.SendAck()
		}
		return err
	default:
		return fmt.Errorf("Unknown manage action '%s' for client '%s'", action.ActionID, action.ClientID)
	}
}

// notification handler invoked when clients have been added, removed or updated
// this invokes a reload of server authn
func (svc *AuthManageClients) onChange() {
	entries := svc.store.GetEntries()
	clients := make([]msgserver.ClientAuthInfo, 0, len(entries))
	for _, e := range entries {
		clients = append(clients, msgserver.ClientAuthInfo{
			ClientID:     e.ClientID,
			ClientType:   e.ClientType,
			PubKey:       e.PubKey,
			PasswordHash: e.PasswordHash,
			Role:         e.Role,
		})
	}
	_ = svc.msgServer.ApplyAuth(clients)
}

// RemoveClient removes a client and disables authentication
func (svc *AuthManageClients) RemoveClient(clientID string) (err error) {
	err = svc.store.Remove(clientID)
	svc.onChange()
	return err
}

// Start subscribes to requests for managing clients
// Register the binding subscription using the given connection
func (svc *AuthManageClients) Start() (err error) {
	if svc.hc != nil {
		svc.mngSub, err = svc.hc.SubServiceCapability(
			auth.AuthManageClientsCapability, svc.HandleActions)
	}
	return err
}

// Stop removes subscriptions
func (svc *AuthManageClients) Stop() {
	if svc.mngSub != nil {
		svc.mngSub.Unsubscribe()
		svc.mngSub = nil
	}
}

func (svc *AuthManageClients) UpdateClient(clientID string, prof auth.ClientProfile) (err error) {
	err = svc.store.Update(clientID, prof)
	return err
}

// NewAuthManageClients creates the capability to manage authentication clients
//
//		store for storing clients
//		msgServer for applying changes to the server
//	 hc hub client for subscribing to receive requests
func NewAuthManageClients(
	store auth.IAuthnStore,
	hc hubclient.IHubClient,
	msgServer msgserver.IMsgServer,
) *AuthManageClients {

	svc := &AuthManageClients{
		store:     store,
		hc:        hc,
		msgServer: msgServer,
	}
	return svc
}
