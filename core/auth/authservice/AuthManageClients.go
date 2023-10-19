package authservice

import (
	"fmt"
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/hubclient"
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
	senderID string, args auth.AddDeviceArgs) (auth.AddDeviceResp, error) {

	//deviceID string, name string, pubKey string) (token string, err error) {
	slog.Info("AddDevice",
		slog.String("deviceID", args.DeviceID),
		slog.String("displayName", args.DisplayName),
		slog.String("pubKey", args.PubKey))

	resp := auth.AddDeviceResp{}
	if args.DeviceID == "" {
		return resp, fmt.Errorf("AddDevice: missing device ID")
	}
	// store/update device.
	err := svc.store.Add(args.DeviceID, auth.ClientProfile{
		ClientID:    args.DeviceID,
		ClientType:  auth.ClientTypeDevice,
		DisplayName: args.DisplayName,
		PubKey:      args.PubKey,
		Role:        auth.ClientRoleDevice,
	})
	if err != nil {
		return resp, err
	}
	err = svc.onChange()

	// generate a device authentication token
	if args.PubKey != "" {
		authInfo := msgserver.ClientAuthInfo{
			ClientID:   args.DeviceID,
			ClientType: auth.ClientTypeDevice,
			PubKey:     args.PubKey,
			Role:       auth.ClientRoleDevice,
		}
		resp.Token, err = svc.msgServer.CreateToken(authInfo)
	}
	return resp, err
}

// AddService adds or updates a service with the admin role
func (svc *AuthManageClients) AddService(
	senderID string, args auth.AddServiceArgs) (auth.AddServiceResp, error) {
	slog.Info("AddService",
		slog.String("senderID", senderID),
		slog.String("serviceID", args.ServiceID),
		slog.String("displayName", args.DisplayName),
		slog.String("pubKey", args.PubKey))

	resp := auth.AddServiceResp{}
	if args.ServiceID == "" {
		return resp, fmt.Errorf("missing service ID")
	}
	err := svc.store.Add(args.ServiceID, auth.ClientProfile{
		ClientID:    args.ServiceID,
		ClientType:  auth.ClientTypeService,
		DisplayName: args.DisplayName,
		PubKey:      args.PubKey,
		Role:        auth.ClientRoleService,
	})
	if err != nil {
		return resp, err
	}
	// generate a service authentication token
	if args.PubKey != "" {
		authInfo := msgserver.ClientAuthInfo{
			ClientID:   args.ServiceID,
			ClientType: auth.ClientTypeService,
			PubKey:     args.PubKey,
			Role:       auth.ClientRoleService,
		}
		resp.Token, err = svc.msgServer.CreateToken(authInfo)
	}
	err = svc.onChange()
	return resp, err
}

// AddUser adds a new user for password authentication
// If a public key is provided a signed token will be returned
func (svc *AuthManageClients) AddUser(
	senderID string, args auth.AddUserArgs) (auth.AddUserResp, error) {
	slog.Info("AddUser",
		slog.String("userID", args.UserID),
		slog.String("displayName", args.DisplayName),
		slog.String("pubKey", args.PubKey),
		slog.String("role", args.Role))

	resp := auth.AddUserResp{}
	if args.UserID == "" {
		return resp, fmt.Errorf("missing user ID")
	}
	err := svc.store.Add(args.UserID, auth.ClientProfile{
		ClientID:    args.UserID,
		ClientType:  auth.ClientTypeUser,
		DisplayName: args.DisplayName,
		PubKey:      args.PubKey,
		Role:        args.Role,
	})
	if err != nil {
		return resp, err
	}
	if args.Password != "" {
		err = svc.store.SetPassword(args.UserID, args.Password)
		if err != nil {
			err = fmt.Errorf("AddUser: user '%s' added, but: %w. Continuing", args.UserID, err)
			slog.Error(err.Error())
		}
	}
	// generate a user token to store
	if args.PubKey != "" {
		authInfo := msgserver.ClientAuthInfo{
			ClientID:   args.UserID,
			ClientType: auth.ClientTypeUser,
			PubKey:     args.PubKey,
			Role:       args.Role,
		}
		resp.Token, err = svc.msgServer.CreateToken(authInfo)
	}
	if err == nil {
		err = svc.onChange()
	}
	return resp, err
}

// GetAuthClientList is for use with the messaging server
func (svc *AuthManageClients) GetAuthClientList() []msgserver.ClientAuthInfo {
	return svc.store.GetAuthClientList()
}

func (svc *AuthManageClients) GetCount() (auth.GetCountResp, error) {
	resp := auth.GetCountResp{}
	resp.N = svc.store.Count()
	return resp, nil
}

// GetClientProfile returns a client's profile
func (svc *AuthManageClients) GetClientProfile(senderID string,
	args auth.GetClientProfileArgs) (auth.GetProfileResp, error) {

	entry, err := svc.store.GetProfile(args.ClientID)
	resp := auth.GetProfileResp{Profile: entry}
	return resp, err
}

// GetProfiles provide a list of known clients and their info.
func (svc *AuthManageClients) GetProfiles() (auth.GetProfilesResp, error) {
	profiles, err := svc.store.GetProfiles()
	resp := auth.GetProfilesResp{Profiles: profiles}
	return resp, err
}

// GetEntries provide a list of known clients and their info including bcrypted passwords
func (svc *AuthManageClients) GetEntries() (entries []auth.AuthnEntry) {
	return svc.store.GetEntries()
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
func (svc *AuthManageClients) RemoveClient(senderID string, args auth.RemoveClientArgs) error {
	err := svc.store.Remove(args.ClientID)
	if err == nil {
		err = svc.onChange()
	}
	return err
}

// Start subscribes to requests for managing clients
// Register the binding subscription using the given connection
func (svc *AuthManageClients) Start() (err error) {
	if svc.hc != nil {
		//svc.mngSub, err = svc.hc.SubRPCRequest(
		//	auth.AuthManageClientsCapability, svc.HandleRequest)
		svc.mngSub, err = hubclient.SubRPCCapability(svc.hc, auth.AuthManageClientsCapability,
			map[string]interface{}{
				auth.AddDeviceMethod:            svc.AddDevice,
				auth.AddServiceMethod:           svc.AddService,
				auth.AddUserMethod:              svc.AddUser,
				auth.GetCountMethod:             svc.GetCount,
				auth.GetClientProfileMethod:     svc.GetClientProfile,
				auth.GetProfilesMethod:          svc.GetProfiles,
				auth.RemoveClientMethod:         svc.RemoveClient,
				auth.UpdateClientMethod:         svc.UpdateClient,
				auth.UpdateClientPasswordMethod: svc.UpdateClientPassword,
				auth.UpdateClientRoleMethod:     svc.UpdateClientRole,
			})
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

func (svc *AuthManageClients) UpdateClient(args auth.UpdateClientArgs) error {
	err := svc.store.Update(args.ClientID, args.Profile)
	return err
}

func (svc *AuthManageClients) UpdateClientPassword(args auth.UpdateClientPasswordArgs) error {
	err := svc.store.SetPassword(args.ClientID, args.Password)
	return err
}

func (svc *AuthManageClients) UpdateClientRole(args auth.UpdateClientRoleArgs) error {
	prof, err := svc.store.GetProfile(args.ClientID)
	if err == nil {
		prof.Role = args.Role
		err = svc.store.Update(args.ClientID, prof)
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
