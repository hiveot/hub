package main

import (
	"flag"
	"fmt"
	"github.com/hiveot/hub/core/auth/authstore"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/runtime/middleware"
	"github.com/hiveot/hub/runtime/protocols"
	"os"
	"time"
)

// Launch the hub digital twin runtime.
// This starts the digital twin stores and the protocol servers.
//
// commandline:  runtime [options]
//
// This runs
func main() {
	flag.Usage = func() {
		fmt.Println("Usage: runtime [options]")
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

	// Initialize the runtime configuration, directories and load keys and certificates
	cfg := runtime.NewRuntimeConfig()
	err := cfg.Setup(&env)

	// setup the middleware chain and its final destination
	authStore := authstore.NewAuthnFileStore(cfg.AuthConfig.StoreDir)
	auth := authn.NewAuthService(&cfg.AuthConfig, authStore)
	directory := directory.NewDirectory(&cfg.DirectoryConfig)
	valueStore := valueStore.NewValueStore(&cfg.ValueStoreConfig)
	mw := middleware.NewMiddleware(&cfg.MiddlewareConfig, auth, directory, valueStore)
	pm := protocols.NewProtocolManager(
		cfg, cfg.ServerKey, cfg.ServerTLS, cfg.CaCert, mw.Handler)

	if err == nil {
		err = auth.Start()
	}
	if err == nil {
		err = directory.Start()
	}
	if err == nil {
		err = valueStore.Start()
	}
	if err == nil {
		err = mw.Start()
	}
	// connect the protocols to the middleware
	if err == nil {
		err = pm.Start()
	}
	//err := hubCfg.Setup(&env, "mqtt", false)
	//if err != nil {
	//	fmt.Println("ERROR:", err.Error())
	//	os.Exit(1)
	//}

	if err != nil {
		println("Runtime startup failed: " + err.Error())
		os.Exit(1)
	}

	// wait until signal
	plugin.WaitForSignal()

	println("Graceful shutdown of Runtime")
	pm.Stop()
	mw.Stop()
	valueStore.Stop()
	directory.Stop()
	auth.Stop()

	// give background tasks time to stop
	time.Sleep(time.Millisecond * 100)
}

//
//// run starts the server and core services
//// This does not return until a signal is received
//func run(cfg *config.HubCoreConfig) error {
//	var err error
//
//	msgServer := service.NewMqttMsgServer(&cfg.MqttServer, authapi.DefaultRolePermissions)
//	err = msgServer.Start()
//	if err != nil {
//		return fmt.Errorf("unable to start server: %w", err)
//	}
//
//	// Start the Auth service mqtt can use either argon2id or brcypt passwords
//	cfg.Auth.Encryption = authapi.PWHASH_BCRYPT
//	authSvc, err := authservice.StartAuthService(cfg.Auth, msgServer, cfg.CaCert)
//	if err != nil {
//		return err
//	}
//
//	// start discovery
//	serverURL, _, _ := msgServer.GetServerURLs()
//	if cfg.EnableMDNS {
//		urlInfo, err := url.Parse(serverURL)
//		if err != nil {
//			return err
//		}
//		port, _ := strconv.Atoi(urlInfo.Port())
//		svc, err := discovery.ServeDiscovery(
//			"mqttcore", "hiveot", urlInfo.Hostname(), port, map[string]string{
//				"rawurl": serverURL,
//				"core":   "mqtt",
//			})
//		_ = svc
//		_ = err
//	}
//
//	// wait until signal
//	fmt.Println("MQTT Hub core started. serverURL=" + serverURL)
//	plugin.WaitForSignal()
//
//	authSvc.Stop()
//	msgServer.Stop()
//	// give background tasks time to stop
//	time.Sleep(time.Millisecond * 100)
//	return nil
//}
