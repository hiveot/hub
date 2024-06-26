package main

import (
	"flag"
	"fmt"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/runtime/transports/mqtttransport/service"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Launch the hub MQTT core
// This starts the embedded messaging service and in-process core services.
//
// commandline:  mqttcore [options]
// Run with '-h' to see the application environment options.
//
// This runs HubCoreConfig.Setup which creates missing directories, certs and
// auth keys and tokens.
func main() {
	flag.Usage = func() {
		fmt.Println("Usage: mqttcore [options]")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
	}
	env := plugin.GetAppEnvironment("", true)
	//env.Core = "mqtt"
	logging.SetLogging(env.LogLevel, "")
	fmt.Println("home: ", env.HomeDir)
	if len(flag.Args()) > 0 {
		println("ERROR: No arguments expected.")
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// setup the core configuration
	hubCfg := config.NewHubCoreConfig()
	err := hubCfg.Setup(&env, "mqtt", false)
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}

	err = run(hubCfg)
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}
}

// run starts the server and core services
// This does not return until a signal is received
func run(cfg *config.HubCoreConfig) error {
	var err error

	msgServer := service.NewMqttMsgServer(&cfg.MqttServer, authapi.DefaultRolePermissions)
	err = msgServer.Start()
	if err != nil {
		return fmt.Errorf("unable to start server: %w", err)
	}

	// Start the Auth service mqtt can use either argon2id or brcypt passwords
	cfg.Auth.Encryption = authapi.PWHASH_BCRYPT
	authSvc, err := authservice.StartAuthService(cfg.Auth, msgServer, cfg.CaCert)
	if err != nil {
		return err
	}

	// start discovery
	serverURL, _, _ := msgServer.GetServerURLs()
	if cfg.EnableMDNS {
		urlInfo, err := url.Parse(serverURL)
		if err != nil {
			return err
		}
		port, _ := strconv.Atoi(urlInfo.Port())
		svc, err := discovery.ServeDiscovery(
			"mqttcore", "hiveot", urlInfo.Hostname(), port, map[string]string{
				"rawurl": serverURL,
				"core":   "mqtt",
			})
		_ = svc
		_ = err
	}

	// wait until signal
	fmt.Println("MQTT Hub core started. serverURL=" + serverURL)
	plugin.WaitForSignal()

	authSvc.Stop()
	msgServer.Stop()
	// give background tasks time to stop
	time.Sleep(time.Millisecond * 100)
	return nil
}
