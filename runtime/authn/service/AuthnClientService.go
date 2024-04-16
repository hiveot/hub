package service

import (
	"crypto/x509"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn"
	"log/slog"
)

// AuthnClientService handles authentication and authorization of the current client
type AuthnClientService struct {
	authnStore api.IAuthnStore
	caCert     *x509.Certificate

	cfg *authn.AuthnConfig

	// key used to create and verify session tokens
	//signingKey keys.IHiveKey

	// the authenticator for jwt tokens
	sessionAuth api.IAuthenticator
}

// GetProfile returns a client's profile
func (svc *AuthnClientService) GetProfile(clientID string) (api.ClientProfile, error) {

	entry, err := svc.authnStore.GetProfile(clientID)
	return entry, err
}

//
//// Start the authentication service.
//// The provided user store must be opened first.
//// This creates accounts for the admin user and launcher if they don't exist.
//func (svc *AuthnClientService) Start() error {
//	slog.Info("starting AuthnService")
//
//	//// before being able to connect, the AuthService and its key must be known
//	//// auth service key is in-memory only
//	//svc.signingKey, err = svc.LoadCreateKeyPair(api.AuthnClientServiceID, svc.cfg.KeysDir)
//	//
//	//if err != nil {
//	//	return nil, err
//	//}
//	//svc.sessionAuth = authenticator.NewJWTAuthenticator(svc.signingKey, svc.authnStore)
//
//	// ensure the password hash algo is valid
//	//if svc.cfg.Encryption != authn.PWHASH_BCRYPT && svc.cfg.Encryption != authn.PWHASH_ARGON2id {
//	//	return fmt.Errorf("Start: Invalid password hash algo: %s", svc.cfg.Encryption)
//	//}
//
//	return svc.sessionAuth, err
//}

// Stop the service, unsubscribe and disconnect from the server
//func (svc *AuthnClientService) Stop() {
//	slog.Info("Stopping AuthnService")
//}

// Login and return a session token
func (svc *AuthnClientService) Login(
	clientID string, password string, sessionID string) (token string, err error) {
	token, err = svc.sessionAuth.Login(clientID, password, sessionID)
	return token, err
}

// RefreshToken requests a new token based on the old token
func (svc *AuthnClientService) RefreshToken(clientID string, oldToken string) (newToken string, err error) {
	prof, err := svc.authnStore.GetProfile(clientID)
	newToken, err = svc.sessionAuth.RefreshToken(clientID, oldToken, prof.TokenValiditySec)
	return newToken, err
}

// ValidateToken verifies that the given token is valid
func (svc *AuthnClientService) ValidateToken(token string) (clientID string, sessionID string, err error) {
	return svc.sessionAuth.ValidateToken(token)
}

func (svc *AuthnClientService) UpdatePassword(clientID string, password string) error {
	slog.Info("SetClientPassword", "clientID", clientID)
	err := svc.authnStore.SetPassword(clientID, password)
	if err != nil {
		slog.Error("Failed changing password", "clientID", clientID, "err", err.Error())
	}
	return err
}
func (svc *AuthnClientService) UpdatePubKey(clientID string, pubKey string) error {
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

// NewAuthnClientService creates an end-user authentication service instance for
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
func NewAuthnClientService(
	authnStore api.IAuthnStore,
	sessionAuth api.IAuthenticator) *AuthnClientService {

	authnSvc := &AuthnClientService{
		authnStore:  authnStore,
		sessionAuth: sessionAuth,
	}
	return authnSvc
}
