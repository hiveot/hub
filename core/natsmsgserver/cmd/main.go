package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/core/auth/authservice"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/natsmsgserver/service"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"gopkg.in/yaml.v3"
	"log/slog"
	"net/url"
	"os"
	"strconv"
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

	env := utils.GetAppEnvironment("", false)
	homeDir := env.HomeDir
	flag.StringVar(&env.HomeDir, "home", env.HomeDir, "Application home directory")
	flag.StringVar(&cfgFile, "c", cfgFile, "Service config file")
	// TODO: move this to hubcli
	flag.BoolVar(&newSetup, "new", newSetup, "Overwrite existing config (use with care!)")
	flag.Usage = func() {
		fmt.Println("Usage: natscore [options] config|run|setup")
		fmt.Println()
		fmt.Println("Options (before command):")
		flag.PrintDefaults()

		fmt.Println("\nCommands:")
		fmt.Println("  (default) run the core services")
		fmt.Println("  config    display configuration")
		fmt.Println("  setup     check and amend the configuration as needed")
		fmt.Println()
	}
	flag.Parse()
	// reload f if home changed
	if homeDir != env.HomeDir {
		env = utils.GetAppEnvironment(env.HomeDir, false)
	}
	env.ServerCore = "nats"
	logging.SetLogging(env.LogLevel, "")

	// setup the core configuration
	hubCfg := config.NewHubCoreConfig()
	err := hubCfg.Setup(env, newSetup)
	cmd := "run"
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

	slog.Info("Starting NATS server")
	msgServer := service.NewNatsMsgServer(&cfg.NatsServer, auth.DefaultRolePermissions)
	err = msgServer.Start()

	if err != nil {
		return fmt.Errorf("unable to start server: %w", err)
	}

	// Start the auth service. NATS requires brcypt passwords
	slog.Info("Starting Auth service")
	cfg.Auth.Encryption = auth.PWHASH_BCRYPT
	authSvc, err := authservice.StartAuthService(cfg.Auth, msgServer)

	// start discovery
	serverURL, _, _ := msgServer.GetServerURLs()
	if cfg.EnableMDNS {
		urlInfo, err := url.Parse(serverURL)
		if err != nil {
			return err
		}
		port, _ := strconv.Atoi(urlInfo.Port())
		svc, err := discovery.ServeDiscovery(
			"natscore", "hiveot", urlInfo.Host, port, map[string]string{
				"rawurl": serverURL,
				"core":   "nats",
			})
		_ = svc
		_ = err
	}

	// wait until signal
	fmt.Println("Hub started. ClientURL=" + serverURL)
	utils.WaitForSignal(context.Background())

	authSvc.Stop()
	msgServer.Stop()
	// give background tasks time to stop
	time.Sleep(time.Millisecond * 100)
	return nil
}
