package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient/connect"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn"
	"log/slog"
	"os"
	"path"
)

// AuthnAdminService handles administration of clients
type AuthnAdminService struct {
	authnStore api.IAuthnStore

	cfg *authn.AuthnConfig

	// key used to create and verify session tokens
	signingKey keys.IHiveKey

	// the authenticator for jwt tokens
	sessionAuth api.IAuthenticator
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
func (svc *AuthnAdminService) AddClient(
	clientType api.ClientType,
	clientID string, displayName string,
	pubKey string, password string) (err error) {

	slog.Info("AddClient",
		slog.String("clientID", clientID),
		slog.String("clientType", string(clientType)))

	if clientType != api.ClientTypeAgent &&
		clientType != api.ClientTypeUser &&
		clientType != api.ClientTypeService {
		err = fmt.Errorf("AddClient: Client type '%s' for client '%s' is not a valid client type",
			clientType, clientID)
		return err
	}
	validitySec := svc.cfg.UserTokenValiditySec
	if clientType == api.ClientTypeService {
		validitySec = svc.cfg.ServiceTokenValiditySec
	} else if clientType == api.ClientTypeAgent {
		validitySec = svc.cfg.AgentTokenValiditySec
	}

	prof, err := svc.authnStore.GetProfile(clientID)
	if err != nil {
		prof = api.ClientProfile{
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
func (svc *AuthnAdminService) AddClientWithTokenFile(
	clientType api.ClientType,
	clientID string, displayName string, validitySec int) error {

	slog.Info("AddClientWithTokenFile", slog.String("clientID", clientID))
	kp, err := keys.LoadCreateKeyPair(clientID, svc.cfg.KeysDir, svc.cfg.DefaultKeyType)
	if err == nil {
		err = svc.AddClient(clientType, clientID, displayName, kp.ExportPublic(), "")
	}
	if err == nil {
		authToken := svc.sessionAuth.CreateSessionToken(clientID, "", validitySec)

		// remove the readonly token file if it exists, to be able to overwrite
		tokenFile := path.Join(svc.cfg.KeysDir, clientID+connect.TokenFileExt)
		_ = os.Remove(tokenFile)
		err = os.WriteFile(tokenFile, []byte(authToken), 0400)
	}
	return err
}

// GetEntries provide a list of known clients
// An entry is a profile with a password hash.
func (svc *AuthnAdminService) GetEntries() (entries []api.AuthnEntry) {
	return svc.authnStore.GetEntries()
}

// GetClientProfile returns a client's profile
func (svc *AuthnAdminService) GetClientProfile(clientID string) (api.ClientProfile, error) {

	entry, err := svc.authnStore.GetProfile(clientID)
	return entry, err
}

// GetProfiles returns a list of all known client profiles
func (svc *AuthnAdminService) GetProfiles() ([]api.ClientProfile, error) {
	profiles, err := svc.authnStore.GetProfiles()
	return profiles, err
}

// RemoveClient removes a client and disables authentication
func (svc *AuthnAdminService) RemoveClient(clientID string) error {
	slog.Info("RemoveClient", "clientID", clientID)
	err := svc.authnStore.Remove(clientID)
	return err
}

// SetRole changes a client's role
func (svc *AuthnAdminService) SetRole(clientID string, role string) error {
	err := svc.authnStore.SetRole(clientID, role)
	return err
}

// Start the authentication service.
// The provided user store must be opened first.
// This creates accounts for the admin user and launcher if they don't exist.
func (svc *AuthnAdminService) Start() error {
	slog.Info("starting AuthnAdminService")

	// ensure the password hash algo is valid
	if svc.cfg.Encryption != authn.PWHASH_BCRYPT && svc.cfg.Encryption != authn.PWHASH_ARGON2id {
		return fmt.Errorf("Start: Invalid password hash algo: %s", svc.cfg.Encryption)
	}

	// Ensure the launcher service and admin user exist and has a saved key and auth token
	launcherID := svc.cfg.LauncherAccountID
	err := svc.AddClientWithTokenFile(api.ClientTypeService,
		launcherID, "Launcher Service", svc.cfg.ServiceTokenValiditySec)
	if err != nil {
		err = fmt.Errorf("failed to setup the launcher account: %w", err)
	}

	// ensure the admin user exists and has a saved key and auth token
	adminID := svc.cfg.AdminAccountID
	err = svc.AddClientWithTokenFile(api.ClientTypeUser,
		adminID, "Administrator", svc.cfg.AgentTokenValiditySec)
	if err != nil {
		err = fmt.Errorf("failed to setup the admin account: %w", err)
	}
	return err
}

// UpdateClientProfile update the client profile.
//
//	profile is the new updated client profile
func (svc *AuthnAdminService) UpdateClientProfile(profile api.ClientProfile) error {
	slog.Info("UpdateClientProfile", slog.String("clientID", profile.ClientID))
	err := svc.authnStore.UpdateProfile(profile.ClientID, profile)
	return err
}

// UpdateClientPassword changes a client's password
func (svc *AuthnAdminService) UpdateClientPassword(clientID string, password string) error {
	slog.Info("UpdateClientPassword", "clientID", clientID)
	err := svc.authnStore.SetPassword(clientID, password)
	if err != nil {
		slog.Error("Failed changing password", "clientID", clientID, "err", err.Error())
	}
	return err
}

// Stop the service, unsubscribe and disconnect from the server
func (svc *AuthnAdminService) Stop() {
	slog.Info("Stopping AuthnService")
	//svc.authnStore.Close()
}

// NewAuthnAdminService creates an authentication service instance for use by administrators only.
//
// The provided store should be opened before calling start and closed after calling stop.
//
//	authnConfig with the configuration settings for the signing key
//	authnStore is the client and credentials store. Must be opened before starting this service.
//	msgServer used to apply changes to users, devices and services
func NewAuthnAdminService(
	authConfig *authn.AuthnConfig,
	authnStore api.IAuthnStore,
	sessionAuth api.IAuthenticator) *AuthnAdminService {

	authnSvc := &AuthnAdminService{
		cfg:         authConfig,
		authnStore:  authnStore,
		sessionAuth: sessionAuth,
	}
	return authnSvc
}
