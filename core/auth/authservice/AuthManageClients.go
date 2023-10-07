package authservice

import (
	"fmt"
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"log/slog"
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
		Role:        auth.ClientRoleDevice,
	})
	if err != nil {
		return "", err
	}
	err = svc.onChange()

	// generate a device authentication token
	if pubKey != "" {
		authInfo := msgserver.ClientAuthInfo{
			ClientID:   deviceID,
			ClientType: auth.ClientTypeDevice,
			PubKey:     pubKey,
			Role:       auth.ClientRoleDevice,
		}
		token, err = svc.msgServer.CreateToken(authInfo)
	}
	return token, err
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
		Role:        auth.ClientRoleService,
	})
	if err != nil {
		return "", err
	}
	// generate a service authentication token
	if pubKey != "" {
		authInfo := msgserver.ClientAuthInfo{
			ClientID:   serviceID,
			ClientType: auth.ClientTypeService,
			PubKey:     pubKey,
			Role:       auth.ClientRoleService,
		}
		token, err = svc.msgServer.CreateToken(authInfo)
	}
	err = svc.onChange()
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
	// generate a user token to store
	if pubKey != "" {
		authInfo := msgserver.ClientAuthInfo{
			ClientID:   userID,
			ClientType: auth.ClientTypeUser,
			PubKey:     pubKey,
			Role:       role,
		}
		token, err = svc.msgServer.CreateToken(authInfo)
	}
	if err == nil {
		err = svc.onChange()
	}
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

// HandleRequest handle incoming RPC requests for managing clients
func (svc *AuthManageClients) HandleRequest(action *hubclient.RequestMessage) error {
	slog.Info("HandleRequest", slog.String("actionID", action.ActionID))

	// TODO: doublecheck the caller is an admin or svc
	switch action.ActionID {
	case auth.AddDeviceReq:
		req := auth.AddDeviceArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		token, err := svc.AddDevice(req.DeviceID, req.DisplayName, req.PubKey)
		if err == nil {
			resp := auth.AddDeviceResp{Token: token}
			reply, _ := ser.Marshal(&resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case auth.AddServiceReq:
		req := auth.AddServiceArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		token, err := svc.AddService(req.ServiceID, req.DisplayName, req.PubKey)
		if err == nil {
			resp := auth.AddServiceResp{Token: token}
			reply, _ := ser.Marshal(&resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case auth.AddUserReq:
		req := auth.AddUserArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		token, err := svc.AddUser(
			req.UserID, req.DisplayName, req.Password, req.PubKey, req.Role)
		if err == nil {
			resp := auth.AddUserResp{Token: token}
			reply, _ := ser.Marshal(&resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case auth.GetCountReq:
		n, err := svc.GetCount()
		resp := auth.GetCountResp{N: n}
		reply, _ := ser.Marshal(&resp)
		err = action.SendReply(reply, nil)
		return err
	case auth.GetProfileReq:
		profile, err := svc.GetProfile(action.ClientID)
		if err == nil {
			resp := auth.GetProfileResp{Profile: profile}
			reply, _ := ser.Marshal(&resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case auth.GetProfilesReq:
		clientList, err := svc.GetProfiles()
		if err == nil {
			resp := auth.GetProfilesResp{Profiles: clientList}
			reply, _ := ser.Marshal(resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case auth.RemoveClientReq:
		req := &auth.RemoveClientArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.RemoveClient(req.ClientID)
		if err == nil {
			err = action.SendAck()
		}
		return err
	case auth.UpdateClientReq:
		req := &auth.UpdateClientArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.UpdateClient(req.ClientID, req.Profile)
		if err == nil {
			err = action.SendAck()
		}
		return err
	case auth.UpdateClientPasswordReq:
		req := &auth.UpdateClientPasswordArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.UpdateClientPassword(req.ClientID, req.Password)
		if err == nil {
			err = action.SendAck()
		}
		return err
	case auth.UpdateClientRoleReq:
		req := &auth.UpdateClientRoleArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.UpdateClientRole(req.ClientID, req.Role)
		if err == nil {
			err = action.SendAck()
		}
		return err
	default:
		return fmt.Errorf("Unknown manage action '%s' for client '%s'", action.ActionID, action.ClientID)
	}
}

// notification handler invoked when clients have been added, removed or updated
// this invokes a reload of server authn
func (svc *AuthManageClients) onChange() error {
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
	err := svc.msgServer.ApplyAuth(clients)
	return err
}

// RemoveClient removes a client and disables authentication
func (svc *AuthManageClients) RemoveClient(clientID string) (err error) {
	err = svc.store.Remove(clientID)
	if err == nil {
		err = svc.onChange()
	}
	return err
}

// Start subscribes to requests for managing clients
// Register the binding subscription using the given connection
func (svc *AuthManageClients) Start() (err error) {
	if svc.hc != nil {
		svc.mngSub, err = svc.hc.SubServiceRPC(
			auth.AuthManageClientsCapability, svc.HandleRequest)
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

func (svc *AuthManageClients) UpdateClientPassword(clientID string, newPassword string) (err error) {
	err = svc.store.SetPassword(clientID, newPassword)
	return err
}

func (svc *AuthManageClients) UpdateClientRole(clientID string, newRole string) (err error) {
	prof, err := svc.store.GetProfile(clientID)
	if err == nil {
		prof.Role = newRole
		err = svc.store.Update(clientID, prof)
	}
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
