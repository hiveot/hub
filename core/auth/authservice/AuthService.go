package authservice

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/auth/authstore"
	"github.com/hiveot/hub/core/auth/config"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
	"os"
	"path"
)

// AuthService handles authentication and authorization requests
type AuthService struct {
	store     authapi.IAuthnStore
	msgServer msgserver.IMsgServer
	caCert    *x509.Certificate

	// the hub client connection to listen to requests
	cfg        config.AuthConfig
	hc         *hubclient.HubClient
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

	// before being able to connect, the AuthService and its key must be known
	core := svc.msgServer.Core()
	tcpAddr, _, udsAddr := svc.msgServer.GetServerURLs()
	svc.hc = hubclient.NewHubClient(tcpAddr, authapi.AuthServiceName, svc.caCert, core)
	myKP, myPubKey := svc.hc.CreateKeyPair()

	// use a temporary instance of the client manager to add itself
	mngClients := NewAuthManageClients(svc.store, nil, svc.msgServer)
	args1 := authapi.AddServiceArgs{
		ServiceID:   authapi.AuthServiceName,
		DisplayName: "Auth Service",
		PubKey:      myPubKey,
	}
	ctx := hubclient.ServiceContext{SenderID: authapi.AuthServiceName}
	resp1, err := mngClients.AddService(ctx, args1)
	if err != nil {
		return fmt.Errorf("failed to setup the auth service: %w", err)
	}

	// nats doesnt support uds?
	_ = udsAddr

	err = svc.hc.ConnectWithToken(myKP, resp1.Token)

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
	svc.msgServer.SetServicePermissions(authapi.AuthServiceName, authapi.AuthManageClientsCapability,
		[]string{authapi.ClientRoleAdmin})
	svc.msgServer.SetServicePermissions(authapi.AuthServiceName, authapi.AuthManageRolesCapability,
		[]string{authapi.ClientRoleAdmin})
	svc.msgServer.SetServicePermissions(authapi.AuthServiceName, authapi.AuthProfileCapability,
		[]string{authapi.ClientRoleViewer, authapi.ClientRoleOperator, authapi.ClientRoleManager, authapi.ClientRoleAdmin})

	// FIXME, what are the permissions for other services like certs, launcher, ...?

	// Ensure the launcher client exists and has a saved key and auth token
	launcherID := svc.cfg.LauncherAccountID
	slog.Info("Start (auth). Adding launcher service", "ID", launcherID)
	_, launcherKeyPub, _ := svc.hc.LoadCreateKeyPair(launcherID, svc.cfg.KeysDir)
	args2 := authapi.AddServiceArgs{
		ServiceID:   launcherID,
		DisplayName: "Launcher Service",
		PubKey:      launcherKeyPub,
	}
	resp2, err := svc.MngClients.AddService(ctx, args2)
	if err == nil {
		// remove the readonly token file if it exists, to be able to overwrite
		tokenFile := path.Join(svc.cfg.KeysDir, launcherID+hubclient.TokenFileExt)
		_ = os.Remove(tokenFile)
		err = os.WriteFile(tokenFile, []byte(resp2.Token), 0400)
	}

	// ensure the admin user exists and has a saved key and auth token
	adminID := svc.cfg.AdminAccountID
	slog.Info("Start (auth). Adding admin user", "ID", adminID)
	_, adminKeyPub, _ := svc.hc.LoadCreateKeyPair(adminID, svc.cfg.KeysDir)
	args3 := authapi.AddUserArgs{
		UserID:      adminID,
		DisplayName: "Administrator",
		PubKey:      adminKeyPub,
		Role:        authapi.ClientRoleAdmin,
	}
	resp3, err := svc.MngClients.AddUser(ctx, args3)
	if err == nil {
		// remove the readonly token file if it exists, to be able to overwrite
		tokenFile := path.Join(svc.cfg.KeysDir, adminID+hubclient.TokenFileExt)
		_ = os.Remove(tokenFile)
		err = os.WriteFile(tokenFile, []byte(resp3.Token), 0400)
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
	store authapi.IAuthnStore, msgServer msgserver.IMsgServer, caCert *x509.Certificate) *AuthService {

	authnSvc := &AuthService{
		caCert:    caCert,
		cfg:       authConfig,
		store:     store,
		msgServer: msgServer,
	}
	return authnSvc
}

// StartAuthService creates and launch the auth service with the given config
// This creates a password store using the config file and password encryption method.
func StartAuthService(cfg config.AuthConfig, msgServer msgserver.IMsgServer, caCert *x509.Certificate) (*AuthService, error) {

	// nats requires bcrypt passwords
	authStore := authstore.NewAuthnFileStore(cfg.PasswordFile, cfg.Encryption)
	authnSvc := NewAuthService(cfg, authStore, msgServer, caCert)
	err := authnSvc.Start()
	if err != nil {
		panic("Cant start Auth service: " + err.Error())
	}
	return authnSvc, err
}
