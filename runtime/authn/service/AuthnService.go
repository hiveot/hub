package service

import (
	"github.com/hiveot/hivehub/runtime/authn/authenticator"
	"github.com/hiveot/hivehub/runtime/authn/authnstore"
	"github.com/hiveot/hivehub/runtime/authn/config"
	"github.com/hiveot/hivehub/runtime/authn/sessions"
	"github.com/hiveot/hivekitgo/messaging"
)

type AuthnService struct {
	SessionAuth messaging.IAuthenticator
	AuthnStore  authnstore.IAuthnStore
	AdminSvc    *AuthnAdminService
	UserSvc     *AuthnUserService
}

// Start the Authentication admin and client services
// This opens the authentication data store and starts the services.
func (svc *AuthnService) Start() error {
	err := svc.AuthnStore.Open()
	if err == nil {
		err = svc.AdminSvc.Start()
		if err == nil {
			err = svc.UserSvc.Start()
		}
	}
	return err
}

// Stop the authentication service and close the store
func (svc *AuthnService) Stop() {
	svc.AdminSvc.Stop()
	svc.UserSvc.Stop()
	svc.AuthnStore.Close()
}

// NewAuthnService creates an instance of the authentication services
func NewAuthnService(
	cfg *config.AuthnConfig,
	authnStore authnstore.IAuthnStore,
	sm *sessions.SessionManager,
	sessionAuth messaging.IAuthenticator) *AuthnService {

	svc := &AuthnService{
		AuthnStore:  authnStore,
		SessionAuth: sessionAuth,
		AdminSvc:    NewAuthnAdminService(cfg, authnStore, sm, sessionAuth),
		UserSvc:     NewAuthnUserService(cfg, authnStore, sessionAuth),
	}
	return svc
}

// StartAuthnService creates and start the authn administration service
// with the given config.
// This creates a password store and authenticator.
//
// authServerURI is the endpoint the service can be reached at to obtain authentication token
// This is provided by the protocol that gives access to the login method.
func StartAuthnService(cfg *config.AuthnConfig) (*AuthnService, error) {

	authnStore := authnstore.NewAuthnFileStore(cfg.PasswordFile, cfg.Encryption)
	//sessionAuth := authenticator.NewJWTAuthenticatorFromFile(
	//	authnStore, cfg.KeysDir, cfg.DefaultKeyType)
	sm := sessions.NewSessionmanager()
	sessionAuth := authenticator.NewPasetoAuthenticatorFromFile(authnStore, cfg.KeysDir, sm)
	sessionAuth.AgentTokenValidityDays = cfg.AgentTokenValidityDays
	sessionAuth.ConsumerTokenValidityDays = cfg.ConsumerTokenValidityDays
	sessionAuth.ServiceTokenValidityDays = cfg.ServiceTokenValidityDays

	svc := NewAuthnService(cfg, authnStore, sm, sessionAuth)
	err := svc.Start()
	return svc, err
}
