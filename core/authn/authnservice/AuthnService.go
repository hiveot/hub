package authnservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/authn/authnstore"
	"github.com/sirupsen/logrus"
)

// AuthnService creates a service handling both manage and user requests.
type AuthnService struct {
	store     auth.IAuthnStore
	msgServer msgserver.IMsgServer

	// the hub client connection to listen to requests
	hc         hubclient.IHubClient
	mngBinding *AuthnManageBinding
	usrBinding *AuthnUserBinding
	// services can be directly used for testing
	MngService *AuthnManage
	UsrService *AuthnUser
}

// Start the service and activate the binding to handle requests
func (svc *AuthnService) Start() (err error) {

	svc.hc, err = svc.msgServer.ConnectInProc(auth.AuthServiceName)
	if err != nil {
		return fmt.Errorf("can't connect authn to server: %w", err)
	}
	svc.MngService = NewAuthnManage(svc.store, svc.msgServer)
	svc.mngBinding = NewAuthnManageBinding(svc.MngService, svc.hc)
	svc.UsrService = NewAuthnUserService(svc.store, svc.msgServer, nil)
	svc.usrBinding = NewAuthnUserBinding(svc.UsrService, svc.hc)

	err = svc.mngBinding.Start()
	if err == nil {
		err = svc.usrBinding.Start()
		if err != nil {
			svc.mngBinding.Stop()
		}
	}
	return err
}

// Stop the service, unsubscribe and disconnect from the server
func (svc *AuthnService) Stop() {
	if svc.mngBinding != nil {
		svc.mngBinding.Stop()
	}
	if svc.usrBinding != nil {
		svc.usrBinding.Stop()
	}
	if svc.hc != nil {
		svc.hc.Disconnect()
	}
}

// NewAuthnService creates an authentication service instance
//
//	store is the client store to store authentication clients
//	msgServer used to apply changes to users, devices and services
func NewAuthnService(store auth.IAuthnStore, msgServer msgserver.IMsgServer) *AuthnService {

	authnSvc := &AuthnService{
		store:     store,
		msgServer: msgServer,
	}
	return authnSvc
}

// StartAuthnService creates and launch the authn service with the given config
// This creates a password store using the config file and password encryption method.
func StartAuthnService(cfg AuthnConfig, msgServer msgserver.IMsgServer) (*AuthnService, error) {

	// nats requires bcrypt passwords
	authStore := authnstore.NewAuthnFileStore(cfg.PasswordFile, cfg.Encryption)
	authnSvc := NewAuthnService(authStore, msgServer)
	err := authnSvc.Start()
	if err != nil {
		logrus.Panicf("cant start test authn service: %s", err)
	}
	return authnSvc, err
}
