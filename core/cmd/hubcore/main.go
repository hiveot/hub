package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"flag"
	"fmt"
	config "github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/hub"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/certsclient"
	"github.com/hiveot/hub/lib/svcconfig"
	"github.com/hiveot/hub/lib/utils"
	"github.com/nats-io/nkeys"
	"gopkg.in/yaml.v3"
	"os"
	"path"
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
		setup(hubCfg, newSetup)
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

	core := hub.NewHubCore(cfg)
	clientURL, err := core.Start()

	if err != nil {
		panic("unable to start server:" + err.Error())
	}

	// wait until signal
	fmt.Println("Hub started. ClientURL=" + clientURL)
	utils.WaitForSignal(context.Background())
}

// Setup creates the missing certificate and key files
// if new is true then overwrite existing files for a setup from scratch
func setup(cfg *config.HubCoreConfig, new bool) {
	var alwaysNewServerCert = false
	var caKey *ecdsa.PrivateKey
	var caCert *x509.Certificate

	newStr := ""
	if new {
		newStr = " with --new"
	}
	fmt.Println("running setup" + newStr)

	// 1.1: handle the CA certificate
	certsDir := path.Dir(cfg.Server.CaCertFile)
	if err := os.MkdirAll(certsDir, 0755); err != nil && err != os.ErrExist {
		errMsg := fmt.Errorf("unable to create certs directory '%s': %w", certsDir, err.Error())
		panic(errMsg)
	}
	if _, err := os.Stat(cfg.Server.CaKeyFile); err == nil && !new {
		caKey, err = certsclient.LoadKeysFromPEM(cfg.Server.CaKeyFile)
		if err == nil {
			caCert, err = certsclient.LoadX509CertFromPEM(cfg.Server.CaCertFile)
		}
		if err != nil {
			panic("unable to CA certificate: " + err.Error())
		}
	} else {
		fmt.Println("creating new CA certificate")
		caCert, caKey, err = certs.CreateCA("hiveot", 365*3)
		if err != nil {
			panic("Unable to create a CA cert: " + err.Error())
		}
		err = certsclient.SaveKeysToPEM(caKey, cfg.Server.CaKeyFile)
		if err == nil {
			err = certsclient.SaveX509CertToPEM(caCert, cfg.Server.CaCertFile)
		}
		// and delete the certs that are based on the key
		_ = os.Remove(cfg.Server.ServerKeyFile)
		_ = os.Remove(cfg.Server.ServerCertFile)
	}
	// 1.2: handle the Server key
	if _, err := os.Stat(cfg.Server.ServerKeyFile); err != nil || new || alwaysNewServerCert {
		// create a new server cert private key and cert
		fmt.Println("creating new server key")
		serverKey := certsclient.CreateECDSAKeys()
		err = certs.SaveKeysToPEM(serverKey, cfg.Server.ServerKeyFile)
		if err != nil {
			panic("Unable to save server key: " + err.Error())
		}
		// with a new key any old cert is useless
		err = os.Remove(cfg.Server.ServerCertFile)
	}
	// 1.3: handle the Server cert
	if _, err := os.Stat(cfg.Server.ServerCertFile); err != nil || new {
		serverKey, _ := certs.LoadKeysFromPEM(cfg.Server.ServerKeyFile)
		// create server certificate file
		fmt.Println("creating new server cert")
		hostName, _ := os.Hostname()
		serverID := "nats-" + hostName
		ou := "hiveot"
		names := []string{"localhost", "127.0.0.1", cfg.Server.Host}
		serverCert, err := certs.CreateServerCert(
			serverID, ou, &serverKey.PublicKey, names, 365, caCert, caKey)
		if err != nil {
			panic("Unable to create a server cert: " + err.Error())
		}
		err = certs.SaveX509CertToPEM(serverCert, cfg.Server.ServerCertFile)
		if err != nil {
			panic("Unable to save server cert")
		}
	}
	// 1.4: make sure the server storage dir exists
	if cfg.Server.DataDir == "" {
		panic("config is missing server data directory")
	}
	if new {
		_ = os.RemoveAll(cfg.Server.DataDir)
	}
	if _, err := os.Stat(cfg.Server.DataDir); err != nil {
		fmt.Println("Creating server data directory: " + cfg.Server.DataDir)
		err = os.MkdirAll(cfg.Server.DataDir, 0750)
	}

	// 1.5: App account key
	if _, err := os.Stat(cfg.Server.AppAccountKeyFile); err != nil {
		fmt.Println("Creating application account key file: " + cfg.Server.AppAccountKeyFile)
		appAcctKey, _ := nkeys.CreateAccount()
		appSeed, _ := appAcctKey.Seed()
		err = os.WriteFile(cfg.Server.AppAccountKeyFile, appSeed, 0400)
	}

	// 2.1: authn directories
	if _, err := os.Stat(cfg.Authn.CertsDir); err != nil {
		_ = os.MkdirAll(cfg.Authn.CertsDir, 0750)
	}
	if _, err := os.Stat(path.Base(cfg.Authn.PasswordFile)); err != nil {
		_ = os.MkdirAll(path.Base(cfg.Authn.PasswordFile), 0750)
	}

	// 3.1: authz directories
	if _, err := os.Stat(cfg.Authz.GroupsDir); err != nil {
		_ = os.MkdirAll(cfg.Authz.GroupsDir, 0750)
	}
	fmt.Println("setup completed successfully")
}
