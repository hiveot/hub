package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/core/auth/authservice"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/natsmsgserver"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

const DefaultCfg = "hub.yaml"

// Launch the hub NATS core
// This starts the embedded messaging service and in-process core services.
//
// commandline:  natscore command options
//
// commands:
//
//	setup    generate missing directories and files.
//	run      start the hub core. core must have been setup before it can run
//	config   show the config that would be used and exit
//
// options:
//
//	--config=file   use the given config file. This defines the folder structure.
//	--home=folder   use the given home folder instead of the base of the application binary
//	--new           use with 'setup', create a brand new setup
//
// setup:
//  1. creates missing folders (see below)
//     if --config=hub.yaml is provided then use this file for the folders
//  2. generate new core system config file(s) in the config folder
//     keep existing files, unless --new is used
//  2. create new self-signed CA and server certificates
//     keep existing files unless --new is used
//  4. initialize the core system storage in the stores folder
//     keep existing storage, unless --new is used
//  5. creates an admin user
//     keep existing admin if any, unless --new is used
//
// setup creates a config file $home/config/hub.yaml with the following folder structure,
// where $home is the directory of the application installation folder:
//
//	$home/bin core application binary
//	$home/plugins contains additional application plugins such as bindings
//	$home/config  configuration files for core and plugins
//	$home/stores  storage of directory and history database
//	$home/certs with server and CA certificates
//
// hub.yaml also defines the pubsub system to use. Currently only nats is supported.
func main() {
	cfgFile := DefaultCfg
	newSetup := false
	logging.SetLogging("info", "")

	f := utils.GetFolders("", false)
	homeDir := f.Home
	flag.StringVar(&f.Home, "home", f.Home, "Application home directory")
	flag.StringVar(&cfgFile, "c", cfgFile, "Service config file")
	flag.BoolVar(&newSetup, "new", newSetup, "Overwrite existing config (use with care!)")
	flag.Usage = func() {
		fmt.Println("Usage: natscore [options] config|run|setup")
		fmt.Println()
		fmt.Println("Options (before command):")
		flag.PrintDefaults()

		fmt.Println("\nCommands:")
		fmt.Println("  config   display configuration")
		fmt.Println("  run      run the core services")
		fmt.Println("  setup    check and amend the configuration as needed")
		fmt.Println()
	}
	flag.Parse()
	// reload f if home changed
	if homeDir != f.Home {
		f = utils.GetFolders(f.Home, false)
	}
	fmt.Println("home: ", f.Home)
	// setup the configuration
	hubCfg := config.NewHubCoreConfig()
	err := hubCfg.Setup(f.Home, cfgFile, newSetup)
	cmd := ""
	if len(flag.Args()) > 0 {
		cmd = flag.Arg(0)
	}

	// only report error if not running setup
	if err != nil && cmd != "setup" {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}

	if cmd == "config" {
		cfgJson, _ := yaml.Marshal(hubCfg)
		fmt.Println("Configuration:\n", string(cfgJson))
	} else if cmd == "setup" {
		//hubCfg.Setup(f.Home, cfgFile, newSetup)
		// already done
	} else if err != nil {
		// do nothing
	} else if cmd == "run" {
		err = run(hubCfg)
	} else {
		err = errors.New("unknown command: " + cmd)
	}
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}
}

// run starts the server and core services
// This does not return until a signal is received
func run(cfg *config.HubCoreConfig) error {
	var err error

	msgServer := natsmsgserver.NewNatsMsgServer(&cfg.NatsServer, auth.DefaultRolePermissions)
	clientURL, err := msgServer.Start()

	if err != nil {
		return fmt.Errorf("unable to start server: %w", err)
	}

	// nats requires brcypt passwords
	cfg.Auth.Encryption = auth.PWHASH_BCRYPT
	authSvc, err := authservice.StartAuthService(cfg.Auth, msgServer)

	// wait until signal
	fmt.Println("Hub started. ClientURL=" + clientURL)
	utils.WaitForSignal(context.Background())

	authSvc.Stop()
	msgServer.Stop()
	// give background tasks time to stop
	time.Sleep(time.Millisecond * 100)
	return nil
}