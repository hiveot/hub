package main

import (
	"context"
	"flag"
	"fmt"
	config "github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/hub"
	"github.com/hiveot/hub/lib/svcconfig"
	"github.com/hiveot/hub/lib/utils"
	"gopkg.in/yaml.v3"
)

const DefaultCfg = "hub.yaml"

// Launch the hub core
// This starts the embedded messaging service and in-process core services.
//
// commandline:  hubcore command options
//
// commands:
//
//	setup    setup the core from scratch. Generate missing certs and account tokens.
//	run      run the hub core. core must have been setup before it can run
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

	f := svcconfig.GetFolders("", false)
	homeDir := f.Home
	flag.StringVar(&f.Home, "home", f.Home, "Application home directory")
	flag.StringVar(&cfgFile, "c", cfgFile, "Service config file")
	flag.BoolVar(&newSetup, "new", newSetup, "Overwrite existing config with setup")
	flag.Usage = func() {
		fmt.Println("Usage: hub [options] config|run|setup\n")
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
		f = svcconfig.GetFolders(f.Home, false)
	}
	hubCfg, err := config.NewHubCoreConfig(f.Home, cfgFile)
	cmd := ""
	if len(flag.Args()) > 0 {
		cmd = flag.Arg(0)
	}
	// only report error if not running setup
	if err != nil && cmd != "setup" {
		fmt.Println("ERROR:", err.Error())
	}

	if cmd == "config" {
		cfgJson, _ := yaml.Marshal(hubCfg)
		fmt.Println("Configuration:\n", string(cfgJson))
	} else if cmd == "setup" {
		hubCfg.Setup(newSetup)
	} else if err != nil {
		// do nothing
	} else if cmd == "run" {
		run(hubCfg)
	} else {
		fmt.Println("unknown command: ", cmd)
	}

}

// run starts the server and core services
// This does not return until a signal is received
func run(cfg *config.HubCoreConfig) {
	var err error

	// run the core and make sure all required certs and folders are available
	cfg.Setup(false)
	core := hub.NewHubCore(cfg)
	clientURL := core.Start()

	if err != nil {
		panic("unable to start server:" + err.Error())
	}

	// wait until signal
	fmt.Println("Hub started. ClientURL=" + clientURL)
	utils.WaitForSignal(context.Background())
}
