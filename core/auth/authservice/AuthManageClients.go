package authservice

import (
	"fmt"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"log/slog"
)

// AuthManageClients handles management of devices,users and service clients
// This implements the IAuthManageClients interface.
type AuthManageClients struct {
	// clients storage
	store authapi.IAuthnStore
	// message server to apply changes to
	msgServer msgserver.IMsgServer
	// messaging client for receiving requests
	hc *hubclient.HubClient
	// subscription to receive requests
	mngSub transports.ISubscription
}

// AddDevice adds an IoT device and generates an authentication token
// This is handled by the underlying messaging core.
func (svc *AuthManageClients) AddDevice(
	ctx hubclient.ServiceContext, args authapi.AddDeviceArgs) (authapi.AddDeviceResp, error) {

	//deviceID string, name string, pubKey string) (token string, err error) {
	slog.Info("AddDevice",
		slog.String("deviceID", args.DeviceID),
		slog.String("displayName", args.DisplayName),
		slog.String("pubKey", args.PubKey))

	resp := authapi.AddDeviceResp{}
	if args.DeviceID == "" {
		return resp, fmt.Errorf("AddDevice: missing device ID")
	}
	// store/update device.
	err := svc.store.Add(args.DeviceID, authapi.ClientProfile{
		ClientID:    args.DeviceID,
		ClientType:  authapi.ClientTypeDevice,
		DisplayName: args.DisplayName,
		PubKey:      args.PubKey,
		Role:        authapi.ClientRoleDevice,
	})
	if err != nil {
		return resp, err
	}
	err = svc.onChange()

	// generate a device authentication token
	if args.PubKey != "" {
		authInfo := msgserver.ClientAuthInfo{
			ClientID:   args.DeviceID,
			ClientType: authapi.ClientTypeDevice,
			PubKey:     args.PubKey,
			Role:       authapi.ClientRoleDevice,
		}
		resp.Token, err = svc.msgServer.CreateToken(authInfo)
	}
	return resp, err
}

// AddService adds or updates a client of type service
func (svc *AuthManageClients) AddService(
	ctx hubclient.ServiceContext, args authapi.AddServiceArgs) (authapi.AddServiceResp, error) {
	slog.Info("AddService",
		slog.String("senderID", ctx.SenderID),
		slog.String("serviceID", args.ServiceID),
		slog.String("displayName", args.DisplayName),
		slog.String("pubKey", args.PubKey))

	resp := authapi.AddServiceResp{}
	if args.ServiceID == "" {
		return resp, fmt.Errorf("missing service ID")
	}
	err := svc.store.Add(args.ServiceID, authapi.ClientProfile{
		ClientID:    args.ServiceID,
		ClientType:  authapi.ClientTypeService,
		DisplayName: args.DisplayName,
		PubKey:      args.PubKey,
		Role:        authapi.ClientRoleService,
	})
	if err != nil {
		return resp, err
	}
	// generate a service authentication token
	if args.PubKey != "" {
		authInfo := msgserver.ClientAuthInfo{
			ClientID:   args.ServiceID,
			ClientType: authapi.ClientTypeService,
			PubKey:     args.PubKey,
			Role:       authapi.ClientRoleService,
		}
		resp.Token, err = svc.msgServer.CreateToken(authInfo)
	}
	err = svc.onChange()
	return resp, err
}

// AddUser adds a new user for password authentication
// If a public key is provided a signed token will be returned
func (svc *AuthManageClients) AddUser(
	ctx hubclient.ServiceContext, args authapi.AddUserArgs) (authapi.AddUserResp, error) {
	slog.Info("AddUser",
		slog.String("senderID", ctx.SenderID),
		slog.String("userID", args.UserID),
		slog.String("displayName", args.DisplayName),
		slog.String("pubKey", args.PubKey),
		slog.String("role", args.Role))

	resp := authapi.AddUserResp{}
	if args.UserID == "" {
		return resp, fmt.Errorf("missing user ID")
	}
	err := svc.store.Add(args.UserID, authapi.ClientProfile{
		ClientID:    args.UserID,
		ClientType:  authapi.ClientTypeUser,
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
			ClientType: authapi.ClientTypeUser,
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

func (svc *AuthManageClients) GetCount() (authapi.GetCountResp, error) {
	resp := authapi.GetCountResp{}
	resp.N = svc.store.Count()
	return resp, nil
}

// GetClientProfile returns a client's profile
func (svc *AuthManageClients) GetClientProfile(ctx hubclient.ServiceContext,
	args authapi.GetClientProfileArgs) (authapi.GetProfileResp, error) {

	entry, err := svc.store.GetProfile(args.ClientID)
	resp := authapi.GetProfileResp{Profile: entry}
	return resp, err
}

// GetProfiles provide a list of known clients and their info.
func (svc *AuthManageClients) GetProfiles() (authapi.GetProfilesResp, error) {
	profiles, err := svc.store.GetProfiles()
	resp := authapi.GetProfilesResp{Profiles: profiles}
	return resp, err
}

// GetEntries provide a list of known clients and their info including bcrypted passwords
func (svc *AuthManageClients) GetEntries() (entries []authapi.AuthnEntry) {
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
func (svc *AuthManageClients) RemoveClient(ctx hubclient.ServiceContext, args authapi.RemoveClientArgs) error {
	slog.Info("RemoveClient", "clientID", args.ClientID)
	err := svc.store.Remove(args.ClientID)
	if err == nil {
		err = svc.onChange()
	}
	return err
}

func (svc *AuthManageClients) SetClientPassword(ctx hubclient.ServiceContext, args authapi.SetClientPasswordArgs) error {
	slog.Info("SetClientPassword", "clientID", args.ClientID)
	err := svc.store.SetPassword(args.ClientID, args.Password)
	return err
}

// Start subscribes to requests for managing clients
// Register the binding subscription using the given connection
func (svc *AuthManageClients) Start() (err error) {
	if svc.hc != nil {
		//svc.mngSub, err = svc.hc.SubRPCRequest(
		//	auth.AuthManageClientsCapability, svc.HandleRequest)
		svc.mngSub, err = svc.hc.SubRPCCapability(authapi.AuthManageClientsCapability,
			map[string]interface{}{
				authapi.AddDeviceMethod:         svc.AddDevice,
				authapi.AddServiceMethod:        svc.AddService,
				authapi.AddUserMethod:           svc.AddUser,
				authapi.GetCountMethod:          svc.GetCount,
				authapi.GetClientProfileMethod:  svc.GetClientProfile,
				authapi.GetProfilesMethod:       svc.GetProfiles,
				authapi.RemoveClientMethod:      svc.RemoveClient,
				authapi.UpdateClientMethod:      svc.UpdateClient,
				authapi.SetClientPasswordMethod: svc.SetClientPassword,
				authapi.UpdateClientRoleMethod:  svc.UpdateClientRole,
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

func (svc *AuthManageClients) UpdateClient(ctx hubclient.ServiceContext, args authapi.UpdateClientArgs) error {
	slog.Info("UpdateClient", "clientID", args.ClientID, "role")
	err := svc.store.Update(args.ClientID, args.Profile)
	return err
}

func (svc *AuthManageClients) UpdateClientRole(ctx hubclient.ServiceContext, args authapi.UpdateClientRoleArgs) error {
	slog.Info("UpdateClientRole", "clientID", args.ClientID, "role", args.Role)
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
	store authapi.IAuthnStore,
	hc *hubclient.HubClient,
	msgServer msgserver.IMsgServer,
) *AuthManageClients {

	svc := &AuthManageClients{
		store:     store,
		hc:        hc,
		msgServer: msgServer,
	}
	return svc
}
