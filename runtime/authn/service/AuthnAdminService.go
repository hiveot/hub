package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/clients"
	authn "github.com/hiveot/hub/runtime/authn/api"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"github.com/hiveot/hub/runtime/authn/config"
	"github.com/hiveot/hub/runtime/authn/sessions"
	authz "github.com/hiveot/hub/runtime/authz/api"
	"log/slog"
	"os"
	"path"
	"time"
)

// AuthnAdminService handles administration of clients
type AuthnAdminService struct {
	authnStore authnstore.IAuthnStore

	cfg *config.AuthnConfig

	// key used to create and verify session tokens
	signingKey keys.IHiveKey

	// the authenticator for auth tokens
	sessionAuth messaging.IAuthenticator

	// sessions
	sm *sessions.SessionManager
}

// AddConsumer adds a consumer account to the service without a role.
// This updates the client info if the client already exists.
//
//	clientID is the ID of the service, agent or user
//	displayName is the friendly name for presentation
//	password is the optional login password. Intended for users if no other credentials are available.
func (svc *AuthnAdminService) AddConsumer(senderID string, args authn.AdminAddConsumerArgs) (err error) {
	//clientID string, displayName string, password string) (err error) {

	slog.Info("AddConsumer", slog.String("clientID", args.ClientID))

	if args.ClientID == "" {
		err = fmt.Errorf("AddConsumer: SenderID is missing")
		return err
	}
	if args.DisplayName == "" {
		args.DisplayName = args.ClientID
	}
	prof, err := svc.authnStore.GetProfile(args.ClientID)
	if err != nil {
		// new profile
		prof = authn.ClientProfile{
			ClientID:    args.ClientID,
			ClientType:  authn.ClientTypeConsumer,
			DisplayName: args.DisplayName,
			//TokenValiditySec: svc.cfg.ConsumerTokenValiditySec,
		}
		err = svc.authnStore.Add(args.ClientID, prof)
		if args.Password != "" {
			err = svc.authnStore.SetPassword(args.ClientID, args.Password)
		}
	} else {
		// client already exists, update password
		//err = fmt.Errorf("Client '%s' already exists", args.ClientID)
		//prof.TokenValiditySec = svc.cfg.ConsumerTokenValiditySec
		if args.Password != "" {
			err = svc.authnStore.SetPassword(args.ClientID, args.Password)
		}
	}
	return err
}

// AddAgent adds or updates a device agent account and assigns the agent role.
// Agents are provided with non-session auth tokens which survive a server restart.
// Agents should store their own key and token files.
func (svc *AuthnAdminService) AddAgent(senderID string,
	args authn.AdminAddAgentArgs) (token string, err error) {

	var prof authn.ClientProfile
	if args.ClientID == "" {
		return token, fmt.Errorf("AddAgent: missing clientID")
	}
	slog.Info("AddAgent", slog.String("agentID", args.ClientID))
	// agents typically create their own key pair
	// services typically don't and have their keys saved on (re)creation
	if args.PubKey == "" {
		kp, err2 := keys.LoadCreateKeyPair(args.ClientID, svc.cfg.KeysDir, keys.KeyTypeEd25519)
		err = err2
		if err == nil {
			args.PubKey = kp.ExportPublic()
		}
	}
	if err == nil {
		// new profile
		prof = authn.ClientProfile{
			ClientID:    args.ClientID,
			ClientType:  authn.ClientTypeAgent,
			DisplayName: args.DisplayName,
			PubKey:      args.PubKey,
			//TokenValiditySec: svc.cfg.AgentTokenValiditySec,
		}
		err = svc.authnStore.Add(args.ClientID, prof)
		if err == nil {
			err = svc.authnStore.SetRole(args.ClientID, string(authz.ClientRoleAgent))
		}
	}
	if err == nil {
		// agent tokens are not restricted to a session. If sessionID matches clientID then
		// no additional session check will take place.
		validity := time.Duration(svc.cfg.AgentTokenValidityDays) * time.Hour * 24
		token = svc.sessionAuth.CreateSessionToken(args.ClientID, args.ClientID, validity)
	}
	return token, err
}

// AddService adds or updates a service account with the service role and key and auth token files.
//
// Notes:
// * Services are provided with non-session auth tokens which survive a server restart.
// * Service keys and tokens are saved in the certs directory under the service name with
// the .key and .token extension.
func (svc *AuthnAdminService) AddService(senderID string,
	args authn.AdminAddServiceArgs) (token string, err error) {

	var prof authn.ClientProfile
	if args.ClientID == "" {
		return token, fmt.Errorf("missing serviceID")
	}
	slog.Info("AddService", slog.String("agentID", args.ClientID))
	// agents typically create their own key pair
	// services typically don't and have their keys saved on (re)creation
	if args.PubKey == "" {
		kp, err2 := keys.LoadCreateKeyPair(args.ClientID, svc.cfg.KeysDir, keys.KeyTypeEd25519)
		err = err2
		if err == nil {
			args.PubKey = kp.ExportPublic()
		}
	}
	if err == nil {
		//tokenValiditySec := svc.cfg.ServiceTokenValiditySec
		// new profile
		prof = authn.ClientProfile{
			ClientID:    args.ClientID,
			ClientType:  authn.ClientTypeService,
			DisplayName: args.DisplayName,
			PubKey:      args.PubKey,
			//TokenValiditySec: tokenValiditySec,
		}
		err = svc.authnStore.Add(args.ClientID, prof)
		if err == nil {
			err = svc.authnStore.SetRole(args.ClientID, string(authz.ClientRoleService))
		}
	}
	if err == nil {
		// service tokens are not linked to a session (sessionID equals clientID)
		validity := time.Duration(svc.cfg.ServiceTokenValidityDays) * time.Hour * 24
		token = svc.sessionAuth.CreateSessionToken(args.ClientID, args.ClientID, validity)

		// remove the readonly token file if it exists, to be able to overwrite
		tokenFile := path.Join(svc.cfg.KeysDir, args.ClientID+clients.TokenFileExt)
		_ = os.Remove(tokenFile)
		err = os.WriteFile(tokenFile, []byte(token), 0400)
	}
	return token, err
}

// GetEntries provide a list of known clients. (internal function)
// An entry is a profile with a password hash.
func (svc *AuthnAdminService) GetEntries() (entries []authnstore.AuthnEntry) {
	return svc.authnStore.GetEntries()
}

// GetClientProfile returns a client's profile
func (svc *AuthnAdminService) GetClientProfile(
	_ string, clientID string) (prof authn.ClientProfile, err error) {

	prof, err = svc.authnStore.GetProfile(clientID)
	return prof, err
}

// GetProfiles returns a list of all known client profiles
func (svc *AuthnAdminService) GetProfiles(
	_ string) (clientProfiles []authn.ClientProfile, err error) {

	profiles, err := svc.authnStore.GetProfiles()
	return profiles, err
}

// GetSessions returns a list of all sessions
func (svc *AuthnAdminService) GetSessions(
	_ string) (sessionResp authn.AdminGetSessionsResp, err error) {

	profiles, err := svc.authnStore.GetProfiles()
	sessionResp = make([]struct {
		ClientID string `json:"clientID,omitempty"`
		Created  string `json:"created,omitempty"`
		Expiry   string `json:"expiry,omitempty"`
	}, len(profiles))
	i := 0
	for _, prof := range profiles {
		sessInfo, found := svc.sm.GetSessionByClientID(prof.ClientID)
		if found {
			sessionResp[i].Expiry = sessInfo.Expiry.Format(utils.MilliTimeFormat)
			sessionResp[i].ClientID = prof.ClientID
			sessionResp[i].Created = sessInfo.Created.Format(utils.MilliTimeFormat)
		}
	}
	return sessionResp, err
}

// NewAgentToken creates a new authentication token for a service or agent.
// This token is not tied to a session so should only be handed out to services or agents
func (svc *AuthnAdminService) NewAgentToken(senderID string, agentID string) (token string, err error) {
	_ = senderID
	prof, err := svc.authnStore.GetProfile(agentID)
	_ = prof
	if err == nil {
		validityDays := 1
		if prof.ClientType == authn.ClientTypeAgent {
			validityDays = svc.cfg.AgentTokenValidityDays
		} else if prof.ClientType == authn.ClientTypeService {
			validityDays = svc.cfg.ServiceTokenValidityDays
		} else {
			validityDays = svc.cfg.ConsumerTokenValidityDays
		}
		validity := time.Duration(validityDays) * time.Hour * 24
		token = svc.sessionAuth.CreateSessionToken(agentID, agentID, validity)
	}
	return token, err
}

// RemoveClient removes a client and disables authentication
func (svc *AuthnAdminService) RemoveClient(senderID string, clientID string) error {
	slog.Info("RemoveClient", "clientID", clientID, "senderID", senderID)
	err := svc.authnStore.Remove(clientID)
	return err
}

// Start the authentication service.
// The provided user store must be opened first.
// This creates accounts for the authn service, the admin user, and launcher if they don't exist.
func (svc *AuthnAdminService) Start() error {
	slog.Info("starting AuthnAdminService")

	// ensure the password hash algo is valid
	if svc.cfg.Encryption != config.PWHASH_BCRYPT && svc.cfg.Encryption != config.PWHASH_ARGON2id {
		return fmt.Errorf("Start: Invalid password hash algo: %s", svc.cfg.Encryption)
	}

	// ensure the authn service exists so authz can verify permissions later on
	authnID := authn.AdminAgentID
	_, err := svc.AddService("", authn.AdminAddServiceArgs{
		ClientID: authnID, DisplayName: "Authn Service", PubKey: ""})

	// Ensure the launcher service and admin user exist and has a saved key and auth token
	launcherID := svc.cfg.LauncherAccountID
	_, err = svc.AddService("", authn.AdminAddServiceArgs{
		ClientID: launcherID, DisplayName: "Launcher Service", PubKey: ""})
	if err != nil {
		err = fmt.Errorf("failed to setup the launcher account: %w", err)
	}

	// ensure the admin user/service exists and has a saved key and auth token
	adminID := svc.cfg.AdminAccountID
	_, err = svc.AddService("", authn.AdminAddServiceArgs{
		ClientID: adminID, DisplayName: "Administrator", PubKey: ""})
	if err != nil {
		err = fmt.Errorf("failed to setup the admin account: %w", err)
	}
	return err
}

// SetClientPassword sets a new client password
func (svc *AuthnAdminService) SetClientPassword(senderID string,
	args authn.AdminSetClientPasswordArgs) error {
	slog.Info("UpdateClientPassword", "clientID", args.ClientID)
	err := svc.authnStore.SetPassword(args.ClientID, args.Password)
	if err != nil {
		slog.Error("Failed changing password", "clientID", args.ClientID, "err", err.Error())
	}
	return err
}

// UpdateClientProfile update the client profile.
//
//	profile is the new updated client profile
func (svc *AuthnAdminService) UpdateClientProfile(
	senderID string, profile authn.ClientProfile) error {

	slog.Info("UpdateClientProfile",
		slog.String("clientID", profile.ClientID),
		slog.String("senderID", senderID))
	err := svc.authnStore.UpdateProfile(profile.ClientID, profile)
	return err
}

// Stop the service, unsubscribe and disconnect from the server
func (svc *AuthnAdminService) Stop() {
	slog.Info("Stopping AuthnService")
	//svc.authnStore.Remove()
}

// NewAuthnAdminService creates an authentication service instance for use by administrators only.
//
// The provided store should be opened before calling start and closed after calling stop.
//
//	authnConfig with the configuration settings for the signing key
//	authnStore is the client and credentials store. Must be opened before starting this service.
//	sm session manager for viewing active client sessions
//	sessionAuth authenticator for new or extended sessions
func NewAuthnAdminService(
	authConfig *config.AuthnConfig,
	authnStore authnstore.IAuthnStore,
	sm *sessions.SessionManager,
	sessionAuth messaging.IAuthenticator) *AuthnAdminService {

	authnSvc := &AuthnAdminService{
		cfg:         authConfig,
		authnStore:  authnStore,
		sm:          sm,
		sessionAuth: sessionAuth,
	}
	return authnSvc
}
