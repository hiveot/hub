package authn

import (
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/api"
	"log/slog"
	"path"
)

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
	// Auth token validity for services in days
	ServiceTokenValiditySec int `yaml:"serviceTokenValiditySec,omitempty"`
	// Auth token validity for consumers in days
	UserTokenValiditySec int `yaml:"userTokenValiditySec,omitempty"`

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
		cfg.PasswordFile = api.DefaultPasswordFile
	}
	if !path.IsAbs(cfg.PasswordFile) {
		cfg.PasswordFile = path.Join(storesDir, "authn", cfg.PasswordFile)
	}

	if cfg.Encryption == "" {
		cfg.Encryption = api.PWHASH_ARGON2id
	}
	if cfg.Encryption != api.PWHASH_BCRYPT && cfg.Encryption != api.PWHASH_ARGON2id {
		slog.Error("unknown password encryption method. Reverting to ARGON2id", "Encoding", cfg.Encryption)
		cfg.Encryption = api.PWHASH_ARGON2id
	}

	if cfg.AgentTokenValiditySec == 0 {
		cfg.AgentTokenValiditySec = api.DefaultAgentTokenValiditySec
	}
	if cfg.ServiceTokenValiditySec == 0 {
		cfg.ServiceTokenValiditySec = api.DefaultServiceTokenValiditySec
	}
	if cfg.UserTokenValiditySec == 0 {
		cfg.UserTokenValiditySec = api.DefaultUserTokenValiditySec
	}
	cfg.KeysDir = keysDir
	cfg.AdminAccountID = api.DefaultAdminUserID
	cfg.LauncherAccountID = api.DefaultLauncherServiceID

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
		Encryption: api.PWHASH_ARGON2id,
	}
	return cfg
}
