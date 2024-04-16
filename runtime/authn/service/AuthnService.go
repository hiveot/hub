package service

import (
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn"
	"github.com/hiveot/hub/runtime/authn/authenticator"
	"github.com/hiveot/hub/runtime/authn/authnstore"
)

type AuthnService struct {
	SessionAuth api.IAuthenticator
	AuthnStore  api.IAuthnStore
	AdminSvc    *AuthnAdminService
	ClientSvc   *AuthnClientService
}

// Start the Authentication admin and client services
// This opens the authentication data store and starts the services.
func (svc *AuthnService) Start() error {
	err := svc.AuthnStore.Open()
	//svc.SessionAuth = authenticator.NewJWTAuthenticator(svc.AuthnStore, signingKey)
	if err == nil {
		err = svc.AdminSvc.Start()
		if err == nil {
			svc.ClientSvc = NewAuthnClientService(svc.AuthnStore, svc.SessionAuth)
		}
	}
	return err
}

// Stop the authentication service and close the store
func (svc *AuthnService) Stop() {
	svc.AdminSvc.Stop()
	//svc.ClientSvc.Stop() // n/a
	svc.AuthnStore.Close()
}

// NewAuthnService creates an instance of the authentication services
// The 'AdminSvc' and 'ClientSvc' can be used directly.
func NewAuthnService(
	cfg *authn.AuthnConfig,
	authnStore api.IAuthnStore,
	sessionAuth api.IAuthenticator) *AuthnService {

	svc := &AuthnService{
		AuthnStore: authnStore,
		AdminSvc:   NewAuthnAdminService(cfg, authnStore, sessionAuth),
		ClientSvc:  nil, // set on start
	}
	return svc
}

// StartAuthnService creates and start the authn administration service
// with the given config.
// This creates a password store and authenticator.
func StartAuthnService(cfg *authn.AuthnConfig) *AuthnService {

	authnStore := authnstore.NewAuthnFileStore(cfg.PasswordFile, cfg.Encryption)
	sessionAuth := authenticator.NewJWTAuthenticatorFromFile(
		authnStore, cfg.KeysDir, cfg.DefaultKeyType)
	svc := NewAuthnService(cfg, authnStore, sessionAuth)
	return svc
}
