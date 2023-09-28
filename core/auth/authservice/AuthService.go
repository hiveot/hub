package authservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/msgserver"
	auth2 "github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/auth/authstore"
	"log/slog"
)

// AuthService handles authentication and authorization requests
type AuthService struct {
	store     auth.IAuthnStore
	msgServer msgserver.IMsgServer

	// the hub client connection to listen to requests
	cfg        auth2.AuthConfig
	hc         hubclient.IHubClient
	MngClients *AuthManageClients
	MngRoles   *AuthManageRoles
	MngProfile *AuthManageProfile
}

// Start the service and activate the binding to handle requests
// This adds an 'auth' service client and an admin user
func (svc *AuthService) Start() (err error) {

	slog.Info("starting AuthService")
	err = svc.store.Open()
	if err != nil {
		return err
	}

	// before being able to connect, the AuthServiceName must be known
	myKey, myKeyPub := svc.msgServer.CreateKP()
	_ = myKey
	err = svc.store.Add(auth.AuthServiceName, auth.ClientProfile{
		ClientID:          auth.AuthServiceName,
		ClientType:        auth.ClientTypeService,
		DisplayName:       "Auth",
		PubKey:            myKeyPub,
		TokenValidityDays: 0,
		Role:              auth.ClientRoleAdmin,
	})
	if err != nil {
		return err
	}
	// setup the server with the initial client list
	err = svc.msgServer.ApplyAuth(svc.store.GetAuthClientList())
	if err != nil {
		return fmt.Errorf("auth failed to setup the server: %w", err)
	}

	svc.hc, err = svc.msgServer.ConnectInProc(auth.AuthServiceName)
	if err != nil {
		return fmt.Errorf("can't connect authn to server: %w", err)
	}
	svc.MngClients = NewAuthManageClients(svc.store, svc.hc, svc.msgServer)
	svc.MngProfile = NewAuthManageProfile(svc.store, nil, svc.hc, svc.msgServer)
	svc.MngRoles = NewAuthManageRoles(svc.store, svc.hc, svc.msgServer)

	err = svc.MngClients.Start()
	if err == nil {
		err = svc.MngRoles.Start()
	}
	if err == nil {
		err = svc.MngProfile.Start()
	}
	if err != nil {
		svc.MngClients.Stop()
		svc.MngRoles.Stop()
		svc.MngProfile.Stop()
		svc.hc.Disconnect()
		return
	}

	// set the client roles required to use the service capabilities
	svc.msgServer.SetServicePermissions(auth.AuthServiceName, auth.AuthManageClientsCapability,
		[]string{auth.ClientRoleAdmin})
	svc.msgServer.SetServicePermissions(auth.AuthServiceName, auth.AuthManageRolesCapability,
		[]string{auth.ClientRoleAdmin})
	svc.msgServer.SetServicePermissions(auth.AuthServiceName, auth.AuthProfileCapability,
		[]string{auth.ClientRoleViewer, auth.ClientRoleOperator, auth.ClientRoleManager, auth.ClientRoleAdmin})

	// ensure the launcher client exists and has a key and service token
	_, launcherKeyPub, _ := svc.MngClients.LoadCreateUserKey(svc.cfg.LauncherKeyFile)
	svc.store.Add(auth.DefaultLauncherServiceID, auth.ClientProfile{
		ClientID:    auth.DefaultLauncherServiceID,
		ClientType:  auth.ClientTypeService,
		DisplayName: "Launcher Service",
		PubKey:      launcherKeyPub,
		// TODO: what mechanism refreshes the launcher token?
		TokenValidityDays: auth.DefaultServiceTokenValidityDays,
		Role:              auth.ClientRoleService,
	})
	_, err = svc.MngClients.LoadCreateUserToken(auth.DefaultLauncherServiceID, svc.cfg.LauncherTokenFile)

	// ensure the admin user exists and has a user token
	_, adminKeyPub, _ := svc.MngClients.LoadCreateUserKey(svc.cfg.AdminUserKeyFile)
	svc.store.Add(auth.DefaultAdminUserID, auth.ClientProfile{
		ClientID:    auth.DefaultAdminUserID,
		ClientType:  auth.ClientTypeUser,
		DisplayName: "Administrator",
		PubKey:      adminKeyPub,
		// TODO: what mechanism refreshes the admin token without a password?
		TokenValidityDays: auth.DefaultUserTokenValidityDays,
		Role:              auth.ClientRoleAdmin,
	})
	_, err = svc.MngClients.LoadCreateUserToken(auth.DefaultAdminUserID, svc.cfg.AdminUserTokenFile)

	return err
}

// Stop the service, unsubscribe and disconnect from the server
func (svc *AuthService) Stop() {
	if svc.MngClients != nil {
		svc.MngClients.Stop()
		svc.MngClients = nil
	}
	if svc.MngProfile != nil {
		svc.MngProfile.Stop()
	}
	if svc.MngRoles != nil {
		svc.MngRoles.Stop()
	}
	if svc.hc != nil {
		svc.hc.Disconnect()
	}
	svc.store.Close()
}

// NewAuthnService creates an authentication service instance
//
//	store is the client store to store authentication clients
//	msgServer used to apply changes to users, devices and services
func NewAuthnService(authConfig auth2.AuthConfig, store auth.IAuthnStore, msgServer msgserver.IMsgServer) *AuthService {

	authnSvc := &AuthService{
		cfg:       authConfig,
		store:     store,
		msgServer: msgServer,
	}
	return authnSvc
}

// StartAuthService creates and launch the auth service with the given config
// This creates a password store using the config file and password encryption method.
func StartAuthService(cfg auth2.AuthConfig, msgServer msgserver.IMsgServer) (*AuthService, error) {

	// nats requires bcrypt passwords
	authStore := authstore.NewAuthnFileStore(cfg.PasswordFile, cfg.Encryption)
	authnSvc := NewAuthnService(cfg, authStore, msgServer)
	err := authnSvc.Start()
	if err != nil {
		panic("Cant start Auth service: " + err.Error())
	}
	return authnSvc, err
}
