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

// AddConsumer adds a consumer account to the service without a role.
// This updates the client info if the client already exists.
//
//	clientID is the ID of the service, agent or user
//	displayName is the friendly name for presentation
//	password is the optional login password. Intended for users if no other credentials are available.
func (svc *AuthnAdminService) AddConsumer(
	clientID string, displayName string, password string) (err error) {

	slog.Info("AddUser", slog.String("clientID", clientID))

	if clientID == "" {
		err = fmt.Errorf("AddClient: ClientID is missing")
		return err
	}
	if displayName == "" {
		displayName = clientID
	}
	prof, err := svc.authnStore.GetProfile(clientID)
	if err != nil {
		// new profile
		prof = api.ClientProfile{
			ClientID:         clientID,
			ClientType:       api.ClientTypeUser,
			DisplayName:      displayName,
			TokenValiditySec: svc.cfg.UserTokenValiditySec,
		}
	} else {
		prof.DisplayName = displayName
		prof.ClientType = api.ClientTypeUser
		prof.DisplayName = displayName
		prof.TokenValiditySec = svc.cfg.UserTokenValiditySec
	}
	err = svc.authnStore.Add(clientID, prof)
	if password != "" {
		err = svc.authnStore.SetPassword(clientID, password)
	}
	return err
}

// AddAgent adds or updates a device agent account and assigns the agent role.
// Agents are provided with non-session auth tokens which survive a server restart.
// Agents should store their own key and token files.
//
//	TODO: use the public key for nonce verification
func (svc *AuthnAdminService) AddAgent(agentID string, displayName string, pubKey string) (
	authToken string, err error) {

	var prof api.ClientProfile
	if agentID == "" {
		return "", fmt.Errorf("Missing agentID")
	}
	slog.Info("AddAgent", slog.String("agentID", agentID))
	// agents typically create their own key pair
	// services typically don't and have their keys saved on (re)creation
	if pubKey == "" {
		kp, err2 := keys.LoadCreateKeyPair(agentID, svc.cfg.KeysDir, svc.cfg.DefaultKeyType)
		err = err2
		if err == nil {
			pubKey = kp.ExportPublic()
		}
	}
	if err == nil {
		// new profile
		prof = api.ClientProfile{
			ClientID:         agentID,
			ClientType:       api.ClientTypeAgent,
			DisplayName:      displayName,
			PubKey:           pubKey,
			TokenValiditySec: svc.cfg.AgentTokenValiditySec,
		}
		err = svc.authnStore.Add(agentID, prof)
		if err == nil {
			err = svc.authnStore.SetRole(agentID, api.ClientRoleAgent)
		}
	}
	if err == nil {
		// agent tokens are not restricted to a session
		authToken = svc.sessionAuth.CreateSessionToken(agentID, "", prof.TokenValiditySec)
	}
	return authToken, err
}

// AddService adds or updates a service account with the service role and key and auth token files.
//
// Services are provided with non-session auth tokens which survive a server restart.
// Service keys and tokens are saved in the certs directory under the service name with
// the .key and .token extension.
//
//	TODO: use the public key for nonce verification
func (svc *AuthnAdminService) AddService(agentID string, displayName string, pubKey string) (
	authToken string, err error) {

	var prof api.ClientProfile
	if agentID == "" {
		return "", fmt.Errorf("missing serviceID")
	}
	slog.Info("AddService", slog.String("agentID", agentID))
	// agents typically create their own key pair
	// services typically don't and have their keys saved on (re)creation
	if pubKey == "" {
		kp, err2 := keys.LoadCreateKeyPair(agentID, svc.cfg.KeysDir, svc.cfg.DefaultKeyType)
		err = err2
		if err == nil {
			pubKey = kp.ExportPublic()
		}
	}
	if err == nil {
		tokenValiditySec := svc.cfg.ServiceTokenValiditySec
		// new profile
		prof = api.ClientProfile{
			ClientID:         agentID,
			ClientType:       api.ClientTypeService,
			DisplayName:      displayName,
			PubKey:           pubKey,
			TokenValiditySec: tokenValiditySec,
		}
		err = svc.authnStore.Add(agentID, prof)
		if err == nil {
			err = svc.authnStore.SetRole(agentID, api.ClientRoleService)
		}
	}
	if err == nil {
		// agent tokens are not restricted to a session
		authToken = svc.sessionAuth.CreateSessionToken(agentID, "", prof.TokenValiditySec)

		// remove the readonly token file if it exists, to be able to overwrite
		tokenFile := path.Join(svc.cfg.KeysDir, agentID+connect.TokenFileExt)
		_ = os.Remove(tokenFile)
		err = os.WriteFile(tokenFile, []byte(authToken), 0400)
	}
	return authToken, err
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

// NewAuthToken creates a new authentication token for a service or agent
// This token is not tied to a session so should only be handed out to services or agents
func (svc *AuthnAdminService) NewAuthToken(clientID string, validitySec int) (token string, err error) {
	prof, err := svc.authnStore.GetProfile(clientID)
	if err == nil {
		if validitySec == 0 {
			validitySec = prof.TokenValiditySec
		}
		if validitySec == 0 {
			validitySec = authn.DefaultAgentTokenValiditySec
		}
		token = svc.sessionAuth.CreateSessionToken(clientID, "", validitySec)
	}
	return token, err
}

// RemoveClient removes a client and disables authentication
func (svc *AuthnAdminService) RemoveClient(clientID string) error {
	slog.Info("RemoveClient", "clientID", clientID)
	err := svc.authnStore.Remove(clientID)
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
	_, err := svc.AddService(launcherID, "Launcher Service", "")
	if err != nil {
		err = fmt.Errorf("failed to setup the launcher account: %w", err)
	}

	// ensure the admin user/service exists and has a saved key and auth token
	adminID := svc.cfg.AdminAccountID
	_, err = svc.AddService(adminID, "Administrator", "")
	if err != nil {
		err = fmt.Errorf("failed to setup the admin account: %w", err)
	}
	return err
}

// SetClientPassword sets a new client password
func (svc *AuthnAdminService) SetClientPassword(clientID string, password string) error {
	slog.Info("UpdateClientPassword", "clientID", clientID)
	err := svc.authnStore.SetPassword(clientID, password)
	if err != nil {
		slog.Error("Failed changing password", "clientID", clientID, "err", err.Error())
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
