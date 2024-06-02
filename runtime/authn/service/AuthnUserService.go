package service

import (
	"crypto/x509"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn"
	"log/slog"
)

// AuthnUserService handles authentication and authorization of regular users
// such as agents, services and end-users.
type AuthnUserService struct {
	authnStore api.IAuthnStore
	caCert     *x509.Certificate

	cfg *authn.AuthnConfig

	// key used to create and verify session tokens
	//signingKey keys.IHiveKey

	// the authenticator for jwt tokens
	sessionAuth api.IAuthenticator
}

// GetProfile returns a client's profile
func (svc *AuthnUserService) GetProfile(clientID string) (api.ClientProfile, error) {

	entry, err := svc.authnStore.GetProfile(clientID)
	return entry, err
}

// Login and return a session token
func (svc *AuthnUserService) Login(
	clientID string, password string) (token string, sessionID string, err error) {
	// a user login always creates a session token
	token, sessionID, err = svc.sessionAuth.Login(clientID, password, sessionID)
	return token, sessionID, err
}

// Start the user facing authentication service.
func (svc *AuthnUserService) Start() error {
	slog.Info("starting AuthnService")
	return nil
}

// Stop the service, unsubscribe and disconnect from the server
func (svc *AuthnUserService) Stop() {
	slog.Info("Stopping AuthnUserService")
}

// RefreshToken requests a new token based on the old token
func (svc *AuthnUserService) RefreshToken(clientID string, oldToken string) (newToken string, err error) {
	//prof, err := svc.authnStore.GetProfile(clientID)
	newToken, err = svc.sessionAuth.RefreshToken(clientID, oldToken)
	return newToken, err
}

// ValidateToken verifies that the given token is valid
func (svc *AuthnUserService) ValidateToken(token string) (clientID string, sessionID string, err error) {
	return svc.sessionAuth.ValidateToken(token)
}

func (svc *AuthnUserService) UpdateName(clientID string, newName string) error {
	slog.Info("UpdateName", "clientID", clientID, "newName", newName)
	prof, err := svc.authnStore.GetProfile(clientID)
	if err == nil {
		prof.DisplayName = newName
		err = svc.authnStore.UpdateProfile(clientID, prof)
	}
	if err != nil {
		slog.Error("Failed changing password", "clientID", clientID, "err", err.Error())
	}
	return err
}
func (svc *AuthnUserService) UpdatePassword(clientID string, password string) error {
	slog.Info("SetClientPassword", "clientID", clientID)
	err := svc.authnStore.SetPassword(clientID, password)
	if err != nil {
		slog.Error("Failed changing password", "clientID", clientID, "err", err.Error())
	}
	return err
}
func (svc *AuthnUserService) UpdatePubKey(clientID string, pubKey string) error {
	slog.Info("UpdatePubKey", "clientID", clientID)
	prof, err := svc.authnStore.GetProfile(clientID)
	if err == nil {
		prof.PubKey = pubKey
		err = svc.authnStore.UpdateProfile(clientID, prof)
	}
	if err != nil {
		slog.Error("Failed updating public key", "clientID", clientID, "err", err.Error())
	}
	return err
}

// NewAuthnUserService creates an end-user authentication service instance for
// logging in and managing a user's own profile. This service is accessible by any
// client.
// This service works in conjunction with the authentication store and the
// session authenticator. The latter is created by the admin service or can be instantiated
// using a signing key.
//
// This service does not have start/stop functions. The authnStore and authenticator
// must be operational before using this service.
//
//	authnStore is the client and credentials store. Must be opened before starting this service.
//	sessionAuth is the authenticator returned by the admin service.
func NewAuthnUserService(
	authnStore api.IAuthnStore,
	sessionAuth api.IAuthenticator) *AuthnUserService {

	authnSvc := &AuthnUserService{
		authnStore:  authnStore,
		sessionAuth: sessionAuth,
	}
	return authnSvc
}
