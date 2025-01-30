package service

import (
	"github.com/hiveot/hub/runtime/authn/authenticator"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"github.com/hiveot/hub/runtime/authn/config"
	"github.com/hiveot/hub/transports"
)

type AuthnService struct {
	SessionAuth transports.IAuthenticator
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
	sessionAuth transports.IAuthenticator) *AuthnService {

	svc := &AuthnService{
		AuthnStore:  authnStore,
		SessionAuth: sessionAuth,
		AdminSvc:    NewAuthnAdminService(cfg, authnStore, sessionAuth),
		UserSvc:     NewAuthnUserService(cfg, authnStore, sessionAuth),
	}
	return svc
}

// StartAuthnService creates and start the authn administration service
// with the given config.
// This creates a password store and authenticator.
func StartAuthnService(cfg *config.AuthnConfig) (*AuthnService, error) {

	authnStore := authnstore.NewAuthnFileStore(cfg.PasswordFile, cfg.Encryption)
	//sessionAuth := authenticator.NewJWTAuthenticatorFromFile(
	//	authnStore, cfg.KeysDir, cfg.DefaultKeyType)
	sessionAuth := authenticator.NewPasetoAuthenticatorFromFile(authnStore, cfg.KeysDir)
	sessionAuth.AgentTokenValiditySec = cfg.AgentTokenValiditySec
	sessionAuth.ConsumerTokenValiditySec = cfg.ConsumerTokenValiditySec
	sessionAuth.ServiceTokenValiditySec = cfg.ServiceTokenValiditySec

	svc := NewAuthnService(cfg, authnStore, sessionAuth)
	err := svc.Start()
	return svc, err
}
