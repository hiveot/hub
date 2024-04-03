package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/bucketstore"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/runtime/authn"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"github.com/hiveot/hub/runtime/authn/service"
	"github.com/hiveot/hub/runtime/authz"
	"github.com/hiveot/hub/runtime/directory"
	service2 "github.com/hiveot/hub/runtime/directory/service"
	"github.com/hiveot/hub/runtime/protocols"
	"github.com/hiveot/hub/runtime/router"
	"github.com/hiveot/hub/runtime/valuestore"
	"log/slog"
	"os"
	"path"
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
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	// startup
	authnSvc, authnStore, err := StartAuthnSvc(&env, &cfg.Authn)
	if err != nil {
		slog.Error("Failed starting authentication service", "err", err)
		os.Exit(1)
	}

	authzSvc, err := StartAuthzSvc(&env, &cfg.Authz, authnStore)
	if err != nil {
		slog.Error("Failed starting authorization service", "err", err)
		os.Exit(1)
	}

	dirSvc, err := StartDirectorySvc(&env, &cfg.Directory)
	if err != nil {
		slog.Error("Failed starting directory service", "err", err)
		os.Exit(1)
	}

	valueSvc, err := StartValueSvc(&env, &cfg.ValueStore)
	if err != nil {
		slog.Error("Failed starting value service", "err", err)
		os.Exit(1)
	}

	routerSvc, err := StartRouterSvc(&cfg.Router)
	if err != nil {
		slog.Error("Failed starting the router service", "err", err)
		os.Exit(1)
	}

	protocolMgr, err := StartProtocolsManager(
		&env, &cfg.Protocols, cfg.ServerKey, cfg.ServerTLS, routerSvc.Handler)
	if err != nil {
		slog.Error("Failed starting the protocol manager", "err", err)
		os.Exit(1)
	}

	// Startup Successful
	// wait until signal
	plugin.WaitForSignal()

	println("Graceful shutdown of Runtime")
	protocolMgr.Stop()
	routerSvc.Stop()
	valueSvc.Stop()
	dirSvc.Stop()
	authzSvc.Stop()
	authnSvc.Stop()

	// give background tasks time to stop
	time.Sleep(time.Millisecond * 100)
}

func StartAuthnSvc(env *plugin.AppEnvironment, cfg *authn.AuthnConfig) (
	svc *service.AuthnService, store authn.IAuthnStore, err error) {

	// setup the Authentication service
	authStore := authnstore.NewAuthnFileStore(cfg.PasswordFile, cfg.Encryption)
	err = authStore.Open()
	if err == nil {
		svc = service.NewAuthnService(cfg, authStore, env.CaCert)
		err = svc.Start()
	}
	return svc, authStore, err
}

func StartAuthzSvc(env *plugin.AppEnvironment, cfg *authz.AuthzConfig,
	authnStore authn.IAuthnStore) (svc *authz.AuthzService, err error) {

	// setup the Authentication service
	svc = authz.NewAuthzService(cfg, authnStore)
	err = svc.Start()
	return svc, err
}

func StartDirectorySvc(env *plugin.AppEnvironment, cfg *directory.DirectoryConfig) (
	svc *service2.DirectoryService, err error) {

	var dirStore buckets.IBucketStore
	if err == nil {
		dirStorePath := path.Join(env.StoresDir, "directory", cfg.StoreFilename)
		dirStore = kvbtree.NewKVStore(dirStorePath)
		err = dirStore.Open()
	}
	if err == nil {
		svc = service2.NewDirectoryService(cfg, dirStore)
		err = svc.Start()
	}
	return svc, err
}

func StartProtocolsManager(
	env *plugin.AppEnvironment, cfg *protocols.ProtocolsConfig,
	serverKey keys.IHiveKey, serverCert *tls.Certificate,
	handler func(tv *things.ThingValue) ([]byte, error)) (
	svc *protocols.ProtocolsManager, err error) {

	pm := protocols.NewProtocolManager(
		cfg, serverKey, serverCert, env.CaCert, handler)

	if err == nil {
		err = pm.Start()
	}
	return pm, err
}

func StartRouterSvc(cfg *router.RouterConfig) (svc *router.RouterService, err error) {
	svc = router.NewRouter(cfg)
	err = svc.Start()
	return svc, err
}

func StartValueSvc(env *plugin.AppEnvironment, cfg *valuestore.ValueStoreConfig) (
	svc *valuestore.ValueService, err error) {

	var valueStore buckets.IBucketStore
	var valueSvc *valuestore.ValueService
	if err == nil {
		storeDir := path.Join(env.StoresDir, "values")
		valueStore = bucketstore.NewBucketStore(storeDir, "values", cfg.StoreFilename)
		err = valueStore.Open()
	}
	if err == nil {
		valueSvc = valuestore.NewThingValueService(cfg, valueStore)
		err = valueSvc.Start()
	}

	if err != nil {
		err = fmt.Errorf("can't open history bucket store: %w", err)
	}

	if err == nil {
		err = svc.Start()
	}
	return svc, err
}
