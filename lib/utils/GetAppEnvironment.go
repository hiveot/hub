package utils

import (
	"crypto/x509"
	"flag"
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// AppEnvironment holds the running environment naming conventions.
// Intended for services and plugins.
// This contains folder locations, CA certificate and application clientID
type AppEnvironment struct {
	// Directories
	BinDir     string `yaml:"binDir"`     // Application binary folder, eg launcher, cli, ...
	PluginsDir string `yaml:"pluginsDir"` // Plugin folder
	HomeDir    string `yaml:"homeDir"`    // Home folder, default this is the parent of bin, config, certs and logs
	ConfigDir  string `yaml:"configDir"`  // Config folder with application and configuration files
	ConfigFile string `yaml:"configFile"` // Application configuration file. Default is clientID.yaml
	CertsDir   string `yaml:"certsDir"`   // Certificates and keys location
	LogsDir    string `yaml:"logsDir"`    // Logging output
	LogLevel   string `yaml:"logLevel"`   // logging level
	RunDir     string `yaml:"runDir"`     // PID and sockets folder.
	StoresDir  string `yaml:"storesDir"`  // Root of the service stores

	// Server
	ServerURL  string `yaml:"serverURL"`  // override server address, empty for auto-detect
	ServerCore string `yaml:"serverCore"` // override core to use, "nats" or "mqtt". empty for auto-detect

	// Credentials
	CaCert    *x509.Certificate `yaml:"-"`         // default cert if loaded
	ClientID  string            `yaml:"clientID"`  // the clientID based on the application binary name
	KeyFile   string            `yaml:"keyFile"`   // client's key pair file location
	TokenFile string            `yaml:"tokenFile"` // client's auth token file location
}

// LoadConfig loads the application .
//
//	configFile is the optional name. Use "" for default {clientID}.yaml
//
// This returns an error if loading or parsing the config file fails.
// Returns nil if the config file doesn't exist or is loaded successfully.
func (env *AppEnvironment) LoadConfig(configFile string, cfg interface{}) error {
	if configFile == "" {
		configFile = env.ConfigFile
	}
	if !path.IsAbs(configFile) {
		configFile = path.Join(env.CertsDir, configFile)
	}
	if _, err := os.Stat(configFile); err != nil {
		slog.Info("Configuration file not found. Ignored.", "configFile", configFile)
		return nil
	}

	cfgData, err := os.ReadFile(configFile)
	if err != nil {
		err = fmt.Errorf("loading config failed: %w", err)
		return err
	} else {
		slog.Info("Loaded configuration file", "configFile", configFile)
		err = yaml.Unmarshal(cfgData, cfg)
	}
	return err
}

// GetAppEnvironment returns the application environment including folders for use by the Hub services.
//
// Optionally parse commandline flags:
//
//	-home  		alternative home directory. Default is the parent folder of the app binary
//	-clientID  	alternative clientID. Default is the application binary name.
//	-config     alternative config directory. Default is home/certs
//	-configFile alternative application config file. Default is {clientID}.yaml
//	-loglevel   debug, info, warning (default), error
//	-server     optional server URL or "" for auto-detect
//	-core       optional server core or "" for auto-detect
//
// The default 'user based' structure is:
//
//		home
//		  |- bin                Core binaries
//	      |- plugins            Plugin binaries
//		  |- config             Service configuration yaml files
//		  |- certs              CA and service certificates
//		  |- logs               Logging output
//		  |- run                PID files and sockets
//		  |- stores
//		      |- {service}      Store for service
//
// The system based folder structure is used when launched from a path starting
// with /usr or /opt:
//
//	/opt/hiveot/bin            Application binaries, cli and launcher
//	/opt/hiveot/plugins        Plugin binaries
//	/etc/hiveot/conf.d         Service configuration yaml files
//	/etc/hiveot/certs          CA and service certificates
//	/var/log/hiveot            Logging output
//	/run/hiveot                PID files and sockets
//	/var/lib/hiveot/{service}  Storage of service
//
// This uses os.Args[0] application path to determine the home directory, which is the
// parent of the application binary.
// The default clientID is based on the binary name using os.Args[0].
//
//	homeDir to override the auto-detected or commandline paths. Use "" for defaults.
//	withFlags parse the commandline flags for -home and -clientID
func GetAppEnvironment(homeDir string, withFlags bool) AppEnvironment {
	var configFile string
	var configDir string
	var binDir string
	var pluginsDir string
	var certsDir string
	var logsDir string
	var runDir string
	var storesDir string
	clientID := path.Base(os.Args[0])
	loglevel := "warning"
	serverURL := ""
	serverCore := ""

	// default home folder is the parent of the core or plugin binary
	if homeDir == "" {
		binDir = filepath.Dir(os.Args[0])
		if !path.IsAbs(binDir) {
			cwd, _ := os.Getwd()
			binDir = path.Join(cwd, binDir)
		}
		homeDir = filepath.Join(binDir, "..")
	}
	if withFlags {
		// handle commandline options
		flag.StringVar(&homeDir, "home", homeDir, "Application home directory")
		flag.StringVar(&configDir, "config", configDir, "Configuration directory")
		flag.StringVar(&configFile, "configFile", configFile, "Configuration file")
		flag.StringVar(&clientID, "clientID", clientID, "Application clientID to authenticate with")
		flag.StringVar(&loglevel, "loglevel", loglevel, "logging level: debug, warning, info, error")
		flag.StringVar(&serverURL, "serverURL", serverURL, "server URL or empty for auto-detect")
		//flag.StringVar(&serverURL, "serverCore", serverCore, "server core, 'mqtt' or 'nats', or empty for auto-detect")
		if flag.Usage == nil {
			flag.Usage = func() {
				fmt.Println("Usage: " + clientID + " [options] ")
				fmt.Println()
				fmt.Println("Options:")
				flag.PrintDefaults()
			}
		}
		flag.Parse()
	}
	if !path.IsAbs(homeDir) {
		cwd, _ := os.Getwd()
		homeDir = path.Join(cwd, homeDir)
	}

	// Try to be smart about whether to use the system structure.
	// If the path starts with /opt or /usr then use
	// the system folder configuration. This might be changed in future if it turns
	// out not to be so smart at all.
	// Future: make this work on windows
	useSystem := strings.HasPrefix(homeDir, "/usr") ||
		strings.HasPrefix(homeDir, "/opt")

	if useSystem {
		homeDir = filepath.Join("/var", "lib", "hiveot")
		binDir = filepath.Join("/opt", "hiveot")
		pluginsDir = filepath.Join(binDir, "plugins")
		configDir = filepath.Join("/etc", "hiveot", "conf.d")
		certsDir = filepath.Join("/etc", "hiveot", "certs")
		logsDir = filepath.Join("/var", "log", "hiveot")
		runDir = filepath.Join("/run", "hiveot")
		storesDir = filepath.Join("/var", "lib", "hiveot")
	} else { // use application parent dir
		//slog.Infof("homeDir is '%s", homeDir)
		binDir = filepath.Join(homeDir, "bin")
		pluginsDir = filepath.Join(homeDir, "plugins")
		certsDir = filepath.Join(homeDir, "certs")
		logsDir = filepath.Join(homeDir, "logs")
		runDir = filepath.Join(homeDir, "run")
		storesDir = filepath.Join(homeDir, "stores")

		if configDir == "" {
			configDir = filepath.Join(homeDir, "config")
		}
	}
	if configFile == "" {
		configFile = path.Join(configDir, clientID+".yaml")
	}
	// load the CA cert if found
	caCertFile := path.Join(certsDir, certs.DefaultCaCertFile)
	caCert, _ := certs.LoadX509CertFromPEM(caCertFile)

	// determine the expected location of the app auth key and token
	tokenFile := path.Join(certsDir, clientID+".token")
	keyFile := path.Join(certsDir, clientID+".key")

	return AppEnvironment{
		BinDir:     binDir,
		PluginsDir: pluginsDir,
		HomeDir:    homeDir,
		ConfigDir:  configDir,
		ConfigFile: configFile,
		CertsDir:   certsDir,
		LogsDir:    logsDir,
		LogLevel:   loglevel,
		RunDir:     runDir,
		StoresDir:  storesDir,
		// default client
		ClientID:   clientID,
		KeyFile:    keyFile,
		TokenFile:  tokenFile,
		CaCert:     caCert,
		ServerURL:  serverURL,
		ServerCore: serverCore,
	}
}
