package service

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/authn"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"github.com/hiveot/hub/runtime/authn/jwtauth"
	"log/slog"
	"os"
	"path"
)

// AuthnServiceID authentication service identifier for persisting keys
const AuthnServiceID = "authn"

// AuthnService handles authentication and authorization requests
type AuthnService struct {
	store  authn.IAuthnStore
	caCert *x509.Certificate

	cfg *authn.AuthnConfig

	// key used to create and verify session tokens
	signingKey keys.IHiveKey
}

// AddClient adds a new client to the service.
// This updates the client info if the client already exists.
//
// The public key is only required for services and agents.
// TODO: use the public key for nonce verification
//
//	clientID is the ID of the service, agent or user
//	clientType is one of ClientTypeAgent, ClientTypeService or ClientTypeUser
//	displayName is the friendly name for presentation
//	pubKey is the client's serialized public key if available. Required for services and agents.
//	password is the optional login password. Intended for users if no other credentials are available.
func (svc *AuthnService) AddClient(
	clientType authn.ClientType,
	clientID string, displayName string,
	pubKey string, password string) (err error) {

	slog.Info("AddClient",
		slog.String("clientID", clientID),
		slog.String("clientType", string(clientType)))

	if clientType != authn.ClientTypeAgent &&
		clientType != authn.ClientTypeUser &&
		clientType != authn.ClientTypeService {
		err = fmt.Errorf("AddClient: Client type '%s' for client '%s' is not a valid client type",
			clientType, clientID)
		return err
	}
	validitySec := svc.cfg.UserTokenValiditySec
	if clientType == authn.ClientTypeService {
		validitySec = svc.cfg.ServiceTokenValiditySec
	} else if clientType == authn.ClientTypeAgent {
		validitySec = svc.cfg.AgentTokenValiditySec
	}

	prof, err := svc.store.GetProfile(clientID)
	if err != nil {
		prof = authn.ClientProfile{
			ClientID:         clientID,
			ClientType:       clientType,
			DisplayName:      displayName,
			PubKey:           pubKey,
			TokenValiditySec: validitySec,
		}
	}
	err = svc.store.Add(clientID, prof)
	if password != "" {
		err = svc.store.SetPassword(clientID, password)
	}
	return err
}

// AddClientWithTokenFile adds or updates a client with key and auth token file.
// Intended for creating service and admin accounts.
func (svc *AuthnService) AddClientWithTokenFile(
	clientType authn.ClientType,
	clientID string, displayName string, validitySec int) error {

	slog.Info("AddClientWithTokenFile", slog.String("clientID", clientID))
	kp, err := svc.LoadCreateKeyPair(clientID, svc.cfg.KeysDir)
	if err == nil {
		err = svc.AddClient(clientType, clientID, displayName, kp.ExportPublic(), "")
	}
	if err == nil {
		authToken, err2 := svc.CreateSessionToken(clientID, "", validitySec)
		err = err2
		if err == nil {
			// remove the readonly token file if it exists, to be able to overwrite
			tokenFile := path.Join(svc.cfg.KeysDir, clientID+hubclient.TokenFileExt)
			_ = os.Remove(tokenFile)
			err = os.WriteFile(tokenFile, []byte(authToken), 0400)
		}
	}
	return err
}

// CreateSessionToken creates a new session token for the client
//
//	clientID is the account ID of a known client
//	sessionID for which this token is valid. Empty session ID allows any session.
//	validitySec is the token validity period or 0 for default based on client type
func (svc *AuthnService) CreateSessionToken(
	clientID string, sessionID string, validitySec int) (string, error) {

	// TODO: add support for nonce challenge with client pubkey
	token, err := jwtauth.CreateSessionToken(
		clientID, sessionID, svc.signingKey, validitySec)
	return token, err
}

// GetClient returns a client's profile
func (svc *AuthnService) GetClient(clientID string) (authn.ClientProfile, error) {

	entry, err := svc.store.GetProfile(clientID)
	return entry, err
}

// GetAllClients returns a list of all known client profiles
func (svc *AuthnService) GetAllClients() ([]authn.ClientProfile, error) {
	profiles, err := svc.store.GetProfiles()
	return profiles, err
}

// GetEntries provide a list of known clients and their info including hashed passwords
func (svc *AuthnService) GetEntries() (entries []authn.AuthnEntry) {
	return svc.store.GetEntries()
}

// LoadCreateKeyPair loads a public/private key pair from file or create it if it doesn't exist
// This will load or create a file <clientID>.key and <clientID>.pub from the keysDir.
//
//	clientID is the client to create the keys for
//	keysDir is the location of the key file
func (svc *AuthnService) LoadCreateKeyPair(clientID, keysDir string) (kp keys.IHiveKey, err error) {
	if keysDir == "" {
		return nil, fmt.Errorf("certs directory must be provided")
	}

	keyFile := path.Join(keysDir, clientID+hubclient.KPFileExt)
	pubFile := path.Join(keysDir, clientID+hubclient.PubKeyFileExt)

	// load key from file
	kp, err = keys.NewKeyFromFile(keyFile)

	if err != nil {
		// no keyfile, create the key
		kp = keys.NewKey(svc.cfg.DefaultKeyType)

		// save the key for future use
		err = kp.ExportPrivateToFile(keyFile)
		err2 := kp.ExportPublicToFile(pubFile)
		if err2 != nil {
			err = err2
		}
	}

	return kp, err
}

// Login with password and generate a session token
// Intended for end-users that want to establish a session.
//
//	clientID is the client to log in
//	password to verify
//	sessionID of the new session
//
// This returns a session token or an error if failed
func (svc *AuthnService) Login(clientID, password, sessionID string) (token string, err error) {

	clientProfile, err := svc.store.VerifyPassword(clientID, password)
	_ = clientProfile
	if err != nil {
		return "", err
	}
	validitySec := clientProfile.TokenValiditySec
	token, err = svc.CreateSessionToken(clientID, sessionID, validitySec)
	return token, err
}

// RefreshToken issues a new session token for the authenticated user.
// This returns a refreshed token that can be used to connect to the messaging server
// the old token must be a valid jwt token belonging to the clientID
func (svc *AuthnService) RefreshToken(clientID string, oldToken string) (token string, err error) {
	// verify the token
	clientProfile, err := svc.store.GetProfile(clientID)
	if err != nil {
		return "", err
	}
	tokenClientID, sessionID, err := jwtauth.DecodeSessionToken(
		oldToken, svc.signingKey.PublicKey(), "", "")
	if tokenClientID != clientID {
		err = fmt.Errorf("RefreshToken:Token client '%s' differs from client '%s'", tokenClientID, clientID)
	}
	if err != nil {
		return "", fmt.Errorf("error validating oldToken of client %s: %w", clientID, err)
	}
	validitySec := clientProfile.TokenValiditySec
	token, err = svc.CreateSessionToken(clientID, sessionID, validitySec)
	if err != nil {
		slog.Warn("RefreshToken",
			"clientID", clientProfile.ClientID, "err", err.Error())
	}
	return token, err
}

// RemoveClient removes a client and disables authentication
func (svc *AuthnService) RemoveClient(clientID string) error {
	slog.Info("RemoveClient", "clientID", clientID)
	err := svc.store.Remove(clientID)
	return err
}

// UpdateClient update the client profile.
// Intended for administrators.
//
//	clientID is the issuer of the request
//	profile is the new updated client profile
func (svc *AuthnService) UpdateClient(clientID string, profile authn.ClientProfile) error {
	slog.Info("UpdateClient", "clientID", profile.ClientID)
	err := svc.store.Update(profile.ClientID, profile)
	return err
}

func (svc *AuthnService) UpdatePassword(clientID string, password string) error {
	slog.Info("SetClientPassword", "clientID", clientID)
	err := svc.store.SetPassword(clientID, password)
	if err != nil {
		slog.Error("Failed changing password", "clientID", clientID, "err", err.Error())
	}
	return err
}
func (svc *AuthnService) ValidateToken(clientID string, token string) error {
	slog.Info("ValidateToken", slog.String("clientID", clientID))
	cid, sid, err := jwtauth.DecodeSessionToken(token, svc.signingKey.PublicKey(), "", "")
	_ = cid
	_ = sid
	if err != nil {
		slog.Error("Failed changing password", "clientID", clientID, "err", err.Error())
	}
	return err
}

// Start the authentication service.
// This opens the user store and creates accounts for the admin user and launcher
// if they don't exist.
func (svc *AuthnService) Start() (err error) {
	slog.Warn("starting AuthnService")
	err = svc.store.Open()
	if err != nil {
		return err
	}

	// before being able to connect, the AuthService and its key must be known
	// auth service key is in-memory only
	svc.signingKey, err = svc.LoadCreateKeyPair(AuthnServiceID, svc.cfg.KeysDir)

	if err != nil {
		return err
	}

	// Ensure the launcher service and admin user exist and has a saved key and auth token
	launcherID := svc.cfg.LauncherAccountID
	err = svc.AddClientWithTokenFile(authn.ClientTypeService,
		launcherID, "Launcher Service", svc.cfg.ServiceTokenValiditySec)
	if err != nil {
		err = fmt.Errorf("failed to setup the launcher account: %w", err)
	}

	// ensure the admin user exists and has a saved key and auth token
	adminID := svc.cfg.AdminAccountID
	err = svc.AddClientWithTokenFile(authn.ClientTypeUser,
		adminID, "Administrator", svc.cfg.AgentTokenValiditySec)
	if err != nil {
		err = fmt.Errorf("failed to setup the admin account: %w", err)
	}
	return err
}

// Stop the service, unsubscribe and disconnect from the server
func (svc *AuthnService) Stop() {
	slog.Warn("Stopping AuthService")
	svc.store.Close()
}

// NewAuthnService creates an authentication service instance
//
//	store is the client store to store authentication clients
//	msgServer used to apply changes to users, devices and services
func NewAuthnService(authConfig *authn.AuthnConfig,
	store authn.IAuthnStore, caCert *x509.Certificate) *AuthnService {

	authnSvc := &AuthnService{
		caCert: caCert,
		cfg:    authConfig,
		store:  store,
	}
	return authnSvc
}

// StartAuthnService creates and launch the authn service with the given config
// This creates a password store using the config file and password encryption method.
func StartAuthnService(cfg *authn.AuthnConfig, caCert *x509.Certificate) (*AuthnService, error) {

	// nats requires bcrypt passwords
	authStore := authnstore.NewAuthnFileStore(cfg.PasswordFile, cfg.Encryption)
	authnSvc := NewAuthnService(cfg, authStore, caCert)
	err := authnSvc.Start()
	if err != nil {
		panic("Cant start Auth service: " + err.Error())
	}
	return authnSvc, err
}
