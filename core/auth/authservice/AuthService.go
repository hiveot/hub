package authservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/auth/authbinding"
	"github.com/hiveot/hub/core/auth/authstore"
)

// AuthService creates a service handling both manage and user requests.
type AuthService struct {
	store     auth.IAuthnStore
	msgServer msgserver.IMsgServer

	// the hub client connection to listen to requests
	hc           hubclient.IHubClient
	mngBinding   *authbinding.AuthClientsBinding
	MngService   *AuthClientsCapability
	rolesBinding *authbinding.AuthRolesBinding
	RolesService *AuthRolesCapability
	usrBinding   *authbinding.AuthProfileBinding
	UsrService   *AuthProfileCapability
}

// Start the service and activate the binding to handle requests
func (svc *AuthService) Start() (err error) {

	err = svc.store.Open()
	if err != nil {
		return err
	}

	svc.hc, err = svc.msgServer.ConnectInProc(auth.AuthServiceName)
	if err != nil {
		return fmt.Errorf("can't connect authn to server: %w", err)
	}
	svc.MngService = NewAuthClientsCapability(svc.store, svc.msgServer)
	svc.mngBinding = authbinding.NewAuthnClientsBinding(svc.MngService, svc.hc)
	svc.UsrService = NewAuthProfileCapability(svc.store, svc.msgServer, nil)
	svc.usrBinding = authbinding.NewAuthProfileBinding(svc.UsrService, svc.hc)
	svc.RolesService = NewAuthRolesCapability(svc.store, svc.msgServer)
	svc.rolesBinding = authbinding.NewAuthRolesBinding(svc.RolesService, svc.hc)

	err = svc.mngBinding.Start()
	if err == nil {
		err = svc.rolesBinding.Start()
	}
	if err == nil {
		err = svc.usrBinding.Start()
	}
	if err != nil {
		svc.mngBinding.Stop()
		svc.usrBinding.Stop()
		svc.rolesBinding.Stop()
		svc.hc.Disconnect()
		return
	}
	// set the roles required to use the capabilities
	svc.msgServer.SetServicePermissions(auth.AuthServiceName, auth.AuthManageClientsCapability,
		[]string{auth.ClientRoleAdmin})
	svc.msgServer.SetServicePermissions(auth.AuthServiceName, auth.AuthManageRolesCapability,
		[]string{auth.ClientRoleAdmin})
	svc.msgServer.SetServicePermissions(auth.AuthServiceName, auth.AuthProfileCapability,
		[]string{auth.ClientRoleViewer, auth.ClientRoleOperator, auth.ClientRoleManager, auth.ClientRoleAdmin})
	return err
}

// Stop the service, unsubscribe and disconnect from the server
func (svc *AuthService) Stop() {
	if svc.mngBinding != nil {
		svc.mngBinding.Stop()
	}
	if svc.usrBinding != nil {
		svc.usrBinding.Stop()
	}
	if svc.rolesBinding != nil {
		svc.rolesBinding.Stop()
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
func NewAuthnService(store auth.IAuthnStore, msgServer msgserver.IMsgServer) *AuthService {

	authnSvc := &AuthService{
		store:     store,
		msgServer: msgServer,
	}
	return authnSvc
}

// StartAuthService creates and launch the auth service with the given config
// This creates a password store using the config file and password encryption method.
func StartAuthService(cfg AuthConfig, msgServer msgserver.IMsgServer) (*AuthService, error) {

	// nats requires bcrypt passwords
	authStore := authstore.NewAuthnFileStore(cfg.PasswordFile, cfg.Encryption)
	authnSvc := NewAuthnService(authStore, msgServer)
	err := authnSvc.Start()
	if err != nil {
		panic("cant start test authn service: " + err.Error())
	}
	return authnSvc, err
}
