package authservice

import (
	"fmt"
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/auth/authstore"
	"github.com/hiveot/hub/core/auth/config"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/hubconnect"
	"log/slog"
	"os"
)

// AuthService handles authentication and authorization requests
type AuthService struct {
	store     auth.IAuthnStore
	msgServer msgserver.IMsgServer

	// the hub client connection to listen to requests
	cfg        config.AuthConfig
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

	// before being able to connect, the AuthService must be known
	myKey, myKeyPub := svc.msgServer.CreateKP()
	_ = myKey

	// use a temporary instance of the client manager to add itself
	mngClients := NewAuthManageClients(svc.store, nil, svc.msgServer)
	token, err := mngClients.AddService(auth.AuthServiceName, "Auth Service", myKeyPub)
	if err != nil {
		return fmt.Errorf("failed to setup the auth service: %w", err)
	}

	//authServiceProfile := auth.ClientProfile{
	//	ClientID:          auth.AuthServiceName,
	//	ClientType:        auth.ClientTypeService,
	//	DisplayName:       "Auth",
	//	PubKey:            myKeyPub,
	//	TokenValidityDays: 0,
	//	Role:              auth.ClientRoleService,
	//}
	//err = svc.store.Add(auth.AuthServiceName,authServiceProfile)
	//if err != nil {
	//	return err
	//}
	//// setup the server with the initial client list
	//err = svc.msgServer.ApplyAuth(svc.store.GetAuthClientList())
	//if err != nil {
	//	return fmt.Errorf("auth failed to setup the server: %w", err)
	//}

	// create the service client to manage clients, roles and user profiles
	//token,err := svc.msgServer.CreateToken(authServiceProfile)
	//svc.hc, err = svc.msgServer.ConnectInProc(auth.AuthServiceName)
	//if err != nil {
	//	return fmt.Errorf("can't connect authn to server: %w", err)
	//}

	// nats doesnt support uds?
	tcpAddr, _, udsAddr := svc.msgServer.GetServerURLs()
	_ = udsAddr
	core := svc.msgServer.Core()
	svc.hc = hubconnect.NewHubClient(tcpAddr, auth.AuthServiceName, myKey, nil, core)
	err = svc.hc.ConnectWithToken(token)
	if err != nil {
		return err
	}
	svc.MngClients = NewAuthManageClients(svc.store, svc.hc, svc.msgServer)
	svc.MngRoles = NewAuthManageRoles(svc.store, svc.hc, svc.msgServer)
	svc.MngProfile = NewAuthManageProfile(svc.store, nil, svc.hc, svc.msgServer)

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

	// FIXME, what are the permissions for other services like certs, launcher, ...?

	// Ensure the launcher client exists and has a key and service token
	slog.Info("Start (auth). Adding launcher user", "keyfile", svc.cfg.LauncherKeyFile)
	_, launcherKeyPub, _ := svc.MngClients.LoadCreateUserKey(svc.cfg.LauncherKeyFile)
	token, err = svc.MngClients.AddService(auth.DefaultLauncherServiceID, "Launcher Service", launcherKeyPub)
	if err == nil {
		// remove the readonly token file if it already exists
		_ = os.Remove(svc.cfg.LauncherTokenFile)
		err = os.WriteFile(svc.cfg.LauncherTokenFile, []byte(token), 0400)
	}

	// ensure the admin user exists and has a user token
	slog.Info("Start (auth). Adding admin user", "keyfile", svc.cfg.AdminUserKeyFile)
	_, adminKeyPub, _ := svc.MngClients.LoadCreateUserKey(svc.cfg.AdminUserKeyFile)
	token, err = svc.MngClients.AddUser(
		auth.DefaultAdminUserID, "Administrator", "", adminKeyPub, auth.ClientRoleAdmin)
	if err == nil {
		// remove the readonly token file if it already exists
		_ = os.Remove(svc.cfg.AdminUserTokenFile)
		err = os.WriteFile(svc.cfg.AdminUserTokenFile, []byte(token), 0400)
	}
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

// NewAuthService creates an authentication service instance
//
//	store is the client store to store authentication clients
//	msgServer used to apply changes to users, devices and services
func NewAuthService(authConfig config.AuthConfig,
	store auth.IAuthnStore, msgServer msgserver.IMsgServer) *AuthService {

	authnSvc := &AuthService{
		cfg:       authConfig,
		store:     store,
		msgServer: msgServer,
	}
	return authnSvc
}

// StartAuthService creates and launch the auth service with the given config
// This creates a password store using the config file and password encryption method.
func StartAuthService(cfg config.AuthConfig, msgServer msgserver.IMsgServer) (*AuthService, error) {

	// nats requires bcrypt passwords
	authStore := authstore.NewAuthnFileStore(cfg.PasswordFile, cfg.Encryption)
	authnSvc := NewAuthService(cfg, authStore, msgServer)
	err := authnSvc.Start()
	if err != nil {
		panic("Cant start Auth service: " + err.Error())
	}
	return authnSvc, err
}
