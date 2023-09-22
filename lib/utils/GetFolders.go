package utils

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type AppFolders struct {
	Bin     string // Application binary folder, eg launcher, cli, ...
	Plugins string // Plugin folder
	Home    string // Home folder, default this is the parent of bin, config, certs and logs
	Config  string // Config folder with application and service yaml configuration files
	//ConfigFile string
	Certs      string // Certificates and keys
	Logs       string // Logging output
	Run        string // PID and sockets folder.
	Stores     string // Root of the service stores
	SocketPath string // default location of service UDS listening socket, if used
}

// GetFolders returns the application folders for use by the Hub.
//
// The default 'user based' structure is:
//
//	home
//	  |- bin                Application cli and launcher binaries
//	      |- plugins        Plugin binaries
//	  |- config             Service configuration yaml files
//	  |- certs              CA and service certificates
//	  |- logs               Logging output
//	  |- run                PID files and sockets
//	  |- stores
//	      |- {service}      Store for service
//
// The system based folder structure is:
//
//	/opt/hiveot/bin            Application binaries, cli and launcher
//	             |-- plugins   Plugin binaries
//	/etc/hiveot/conf.d         Service configuration yaml files
//	/etc/hiveot/certs          CA and service certificates
//	/var/log/hiveot            Logging output
//	/run/hiveot                PID files and sockets
//	/var/lib/hiveot/{service}  Storage of service
//
// This uses os.Args[0] application path to determine the services folder. The home folder is two
// levels up from the services folder.
//
//	homeFolder is optional in order to override the auto detected paths. Use "" for defaults.
func GetFolders(homeFolder string, useSystem bool) AppFolders {
	// note that filepath should support windows
	if homeFolder == "" {
		cwd := filepath.Dir(os.Args[0])
		if strings.HasSuffix(cwd, "plugins") {
			homeFolder = filepath.Join(cwd, "..")
		} else if strings.HasSuffix(cwd, "bin") {
			homeFolder = filepath.Join(cwd, "..")
		} else {
			// not sure where home is. For now use the parent
			homeFolder = filepath.Join(cwd, "..")
		}
	}
	//slog.Infof("homeFolder is '%s", homeFolder)
	binFolder := filepath.Join(homeFolder, "bin")
	pluginsFolder := filepath.Join(homeFolder, "plugins")
	configFolder := filepath.Join(homeFolder, "config")
	certsFolder := filepath.Join(homeFolder, "certs")
	logsFolder := filepath.Join(homeFolder, "logs")
	runFolder := filepath.Join(homeFolder, "run")
	storesFolder := filepath.Join(homeFolder, "stores")

	if useSystem {
		homeFolder = filepath.Join("/var", "lib", "hiveot")
		binFolder = filepath.Join("/opt", "hiveot")
		pluginsFolder = filepath.Join(binFolder, "plugins")
		configFolder = filepath.Join("/etc", "hiveot", "conf.d")
		certsFolder = filepath.Join("/etc", "hiveot", "certs")
		logsFolder = filepath.Join("/var", "log", "hiveot")
		runFolder = filepath.Join("/run", "hiveot")
		storesFolder = filepath.Join("/var", "lib", "hiveot")
	}

	return AppFolders{
		Bin:     binFolder,
		Plugins: pluginsFolder,
		Home:    homeFolder,
		Config:  configFolder,
		Certs:   certsFolder,
		Logs:    logsFolder,
		Run:     runFolder,
		Stores:  storesFolder,
	}
}

// LoadConfig loads a configuration file from the config folder
// This returns an error if loading or parsing the config file fails.
// Returns nil if the config file doesn't exist or is loaded successfully.
func (f *AppFolders) LoadConfig(name string, cfg interface{}) error {
	configFile := path.Join(f.Config, name)
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
