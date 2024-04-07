package service

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/authn"
	"github.com/hiveot/hub/runtime/authn/authenticator"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"log/slog"
	"os"
	"path"
)

// AuthnServiceID authentication service identifier for persisting keys
const AuthnServiceID = "authn"

// AuthnService handles authentication and authorization requests
type AuthnService struct {
	authnStore authn.IAuthnStore
	caCert     *x509.Certificate

	cfg *authn.AuthnConfig

	// key used to create and verify session tokens
	signingKey keys.IHiveKey

	// the authenticator for jwt tokens
	sessionAuth authn.IAuthenticator
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

	prof, err := svc.authnStore.GetProfile(clientID)
	if err != nil {
		prof = authn.ClientProfile{
			ClientID:         clientID,
			ClientType:       clientType,
			DisplayName:      displayName,
			PubKey:           pubKey,
			TokenValiditySec: validitySec,
		}
	}
	err = svc.authnStore.Add(clientID, prof)
	if password != "" {
		err = svc.authnStore.SetPassword(clientID, password)
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
		authToken, err2 := svc.sessionAuth.CreateSessionToken(clientID, "", validitySec)
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

// GetAllProfiles returns a list of all known client profiles
func (svc *AuthnService) GetAllProfiles() ([]authn.ClientProfile, error) {
	profiles, err := svc.authnStore.GetProfiles()
	return profiles, err
}

// GetProfile returns a client's profile
func (svc *AuthnService) GetProfile(clientID string) (authn.ClientProfile, error) {

	entry, err := svc.authnStore.GetProfile(clientID)
	return entry, err
}

// GetEntries provide a list of known clients
// An entry is a profile with a password hash.
func (svc *AuthnService) GetEntries() (entries []authn.AuthnEntry) {
	return svc.authnStore.GetEntries()
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
		if err == nil {
			err = kp.ExportPublicToFile(pubFile)
		}
	}

	return kp, err
}

// Login with password and generate a session token
// Intended for end-users that want to establish a session.
//
//	clientID is the client to log in
//	password to verify
//	sessionID of the new session or "" to create a new session
//
// This returns a session token or an error if failed
func (svc *AuthnService) Login(clientID, password, sessionID string) (token string, err error) {

	clientProfile, err := svc.authnStore.VerifyPassword(clientID, password)
	_ = clientProfile
	if err != nil {
		return "", err
	}
	validitySec := clientProfile.TokenValiditySec
	token, err = svc.sessionAuth.CreateSessionToken(clientID, sessionID, validitySec)
	return token, err
}

// RemoveClient removes a client and disables authentication
func (svc *AuthnService) RemoveClient(clientID string) error {
	slog.Info("RemoveClient", "clientID", clientID)
	err := svc.authnStore.Remove(clientID)
	return err
}

// UpdateClient update the client profile.
// Intended for administrators.
//
//	senderID is the issuer of the request
//	profile is the new updated client profile
func (svc *AuthnService) UpdateClient(senderID string, profile authn.ClientProfile) error {
	slog.Info("UpdateClient",
		slog.String("clientID", profile.ClientID),
		slog.String("senderID", senderID))
	err := svc.authnStore.UpdateProfile(profile.ClientID, profile)
	return err
}

func (svc *AuthnService) UpdatePassword(clientID string, password string) error {
	slog.Info("SetClientPassword", "clientID", clientID)
	err := svc.authnStore.SetPassword(clientID, password)
	if err != nil {
		slog.Error("Failed changing password", "clientID", clientID, "err", err.Error())
	}
	return err
}

// Start the authentication service.
// The provided user store must be opened first.
// This creates accounts for the admin user and launcher if they don't exist.
func (svc *AuthnService) Start() (sessionAuth authn.IAuthenticator, err error) {
	slog.Info("starting AuthnService")
	//err = svc.authnStore.Open()
	if err != nil {
		return nil, err
	}

	// before being able to connect, the AuthService and its key must be known
	// auth service key is in-memory only
	svc.signingKey, err = svc.LoadCreateKeyPair(AuthnServiceID, svc.cfg.KeysDir)

	if err != nil {
		return nil, err
	}
	svc.sessionAuth = authenticator.NewJWTAuthenticator(svc.signingKey, svc.authnStore)

	// ensure the password hash algo is valid
	if svc.cfg.Encryption != authn.PWHASH_BCRYPT && svc.cfg.Encryption != authn.PWHASH_ARGON2id {
		return nil, fmt.Errorf("Start: Invalid password hash algo: %s", svc.cfg.Encryption)
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
	return svc.sessionAuth, err
}

// Stop the service, unsubscribe and disconnect from the server
func (svc *AuthnService) Stop() {
	slog.Info("Stopping AuthnService")
	//svc.authnStore.Close()
}

// NewAuthnService creates an authentication service instance.
// The provided store will be opened on start and closed on stop.
//
//	authnConfig with the configuration settings
//	authnStore is the client and credentials store. Must be opened before starting this service.
//	msgServer used to apply changes to users, devices and services
func NewAuthnService(
	authConfig *authn.AuthnConfig,
	authnStore authn.IAuthnStore,
	caCert *x509.Certificate) *AuthnService {

	authnSvc := &AuthnService{
		caCert:     caCert,
		cfg:        authConfig,
		authnStore: authnStore,
		// jwtAuthenticator is initialized on startup
	}
	return authnSvc
}

// StartAuthnService creates and launch the authn service with the given config
// This creates a password store using the config file and password encryption method.
// To shut down, stop the service first then close the store.
func StartAuthnService(cfg *authn.AuthnConfig, caCert *x509.Certificate) (
	*AuthnService, authn.IAuthnStore, authn.IAuthenticator, error) {

	// nats requires bcrypt passwords
	authnStore := authnstore.NewAuthnFileStore(cfg.PasswordFile, cfg.Encryption)
	err := authnStore.Open()
	if err != nil {
		return nil, nil, nil, err
	}
	authnSvc := NewAuthnService(cfg, authnStore, caCert)
	sessionAuth, err := authnSvc.Start()
	return authnSvc, authnStore, sessionAuth, err
}
