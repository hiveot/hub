package config

import (
	"github.com/hiveot/hub/lib/keys"
	"log/slog"
	"path"
)

// Session token validity for client types
const (
	DefaultAgentTokenValiditySec    = 90 * 24 * 3600  // 90 days
	DefaultConsumerTokenValiditySec = 30 * 24 * 3600  // 30 days
	DefaultServiceTokenValiditySec  = 365 * 24 * 3600 // 1 year
)

// supported password hashes
const (
	PWHASH_ARGON2id = "argon2id"
	PWHASH_BCRYPT   = "bcrypt" // fallback in case argon2id cannot be used
)

// DefaultAdminUserID is the client ID of the default CLI administrator account
const DefaultAdminUserID = "admin"

// DefaultLauncherServiceID is the client ID of the launcher service
// auth creates a key and auth token for the launcher on startup
const DefaultLauncherServiceID = "launcher"

// DefaultPasswordFile is the recommended password filename for Hub authentication
const DefaultPasswordFile = "hub.passwd"

// AuthnConfig contains the auth service configuration
type AuthnConfig struct {
	// PasswordFile with the file based password store.
	// Use a relative path for using the default $HOME/stores/authn location
	// Use "" for default defined in 'authnstore.DefaultPasswordFile'
	PasswordFile string `yaml:"passwordFile,omitempty"`
	// Encryption of passwords: "argon2id" (default) or "bcrypt"
	Encryption string `yaml:"encryption,omitempty"`

	// Auth token validity for agents in seconds
	AgentTokenValiditySec int `yaml:"agentTokenValiditySec,omitempty"`
	// Auth token validity for consumers in days
	ConsumerTokenValiditySec int `yaml:"consumerTokenValiditySec,omitempty"`
	// Auth token validity for services in days
	ServiceTokenValiditySec int `yaml:"serviceTokenValiditySec,omitempty"`

	// NoAutoStart prevents the auth service for auto starting. Intended for testing or custom implementation.
	NoAutoStart bool `yaml:"noAutoStart,omitempty"`

	// predefined accounts
	// Location of client keys and tokens
	DefaultKeyType    keys.KeyType `yaml:"defaultKeyType,omitempty"` // keys.KeyTypeECDSA
	KeysDir           string       `yaml:"certsDir,omitempty"`
	AdminAccountID    string       `yaml:"adminAccountID,omitempty"`
	LauncherAccountID string       `yaml:"launcherAccountID,omitempty"`
	//AdminUserKeyFile   string `yaml:"adminUserKeyFile,omitempty"`   // default: admin.key
	//AdminUserTokenFile string `yaml:"adminUserTokenFile,omitempty"` // default: admin.token
	//
	//// Setup for an launcher account
	//LauncherKeyFile   string `yaml:"launcherKeyFile,omitempty"`   // default: launcher.key
	//LauncherTokenFile string `yaml:"launcherTokenFile,omitempty"` // default: launcher.token
}

// Setup ensures config is valid
//
//	storesDir is the default storage root directory ($HOME/stores)
func (cfg *AuthnConfig) Setup(keysDir, storesDir string) {

	if cfg.PasswordFile == "" {
		cfg.PasswordFile = DefaultPasswordFile
	}
	if !path.IsAbs(cfg.PasswordFile) {
		cfg.PasswordFile = path.Join(storesDir, "authn", cfg.PasswordFile)
	}

	if cfg.Encryption == "" {
		cfg.Encryption = PWHASH_ARGON2id
	}
	if cfg.Encryption != PWHASH_BCRYPT && cfg.Encryption != PWHASH_ARGON2id {
		slog.Error("unknown password encryption method. Reverting to ARGON2id", "Encoding", cfg.Encryption)
		cfg.Encryption = PWHASH_ARGON2id
	}

	if cfg.AgentTokenValiditySec == 0 {
		cfg.AgentTokenValiditySec = DefaultAgentTokenValiditySec
	}
	if cfg.ServiceTokenValiditySec == 0 {
		cfg.ServiceTokenValiditySec = DefaultServiceTokenValiditySec
	}
	if cfg.ConsumerTokenValiditySec == 0 {
		cfg.ConsumerTokenValiditySec = DefaultConsumerTokenValiditySec
	}
	cfg.KeysDir = keysDir
	cfg.AdminAccountID = DefaultAdminUserID
	cfg.LauncherAccountID = DefaultLauncherServiceID

	//if cfg.AdminUserKeyFile == "" {
	//	cfg.AdminUserKeyFile = .DefaultAdminUserID + ".key"
	//}
	//if !path.IsAbs(cfg.AdminUserKeyFile) {
	//	cfg.AdminUserKeyFile = path.Join(keysDir, cfg.AdminUserKeyFile)
	//}
	//
	//if cfg.AdminUserTokenFile == "" {
	//	cfg.AdminUserTokenFile = .DefaultAdminUserID + ".token"
	//}
	//if !path.IsAbs(cfg.AdminUserTokenFile) {
	//	cfg.AdminUserTokenFile = path.Join(keysDir, cfg.AdminUserTokenFile)
	//}
	//
	//if cfg.LauncherKeyFile == "" {
	//	cfg.LauncherKeyFile = .DefaultLauncherServiceID + ".key"
	//}
	//if !path.IsAbs(cfg.LauncherKeyFile) {
	//	cfg.LauncherKeyFile = path.Join(keysDir, cfg.LauncherKeyFile)
	//}
	//if cfg.LauncherTokenFile == "" {
	//	cfg.LauncherTokenFile = .DefaultLauncherServiceID + ".token"
	//}
	//if !path.IsAbs(cfg.LauncherTokenFile) {
	//	cfg.LauncherTokenFile = path.Join(keysDir, cfg.LauncherTokenFile)
	//}
}

func NewAuthnConfig() AuthnConfig {
	cfg := AuthnConfig{
		// key to use for creating keys
		DefaultKeyType: keys.KeyTypeECDSA,
		// default password encryption method
		Encryption: PWHASH_ARGON2id,
	}
	return cfg
}
