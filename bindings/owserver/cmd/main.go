package main

import (
	"path"

	"github.com/hiveot/hivekit/go/modules/factory"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hub/bindings/owserver/config"
	"github.com/hiveot/hub/bindings/owserver/service"
	"github.com/hiveot/hub/lib/plugin"
)

// Recipe for a simple device that provides a standalone server for a counter module.

var mods = map[string]factory.ModuleDefinition{
	client.NewClientModuleType: {
		Constructor: client.NewClientFactory,
	},
	// owserver is the application module
	owserver.OwServerBindingModuleType: {
		Constructor: owserver.OWServerBindingFactory,
	},
}

// Start the OWServer protocol binding
func main() {
	// TODO: migrate to use the factory with a recipe
	env := factory.NewAppEnvironment("", true)
	factory := factorypkg.NewModuleFactory(env, mods)
	err := factory.Start()
	serviceID := env.AppID
	cfg := config.NewConfig()
	_ = env.LoadConfig(&cfg)
	storePath := path.Join(env.StoresDir, env.AppID)
	binding := service.NewOWServerBinding(serviceID, storePath, cfg)
	plugin.StartPlugin(binding, serviceID, env.CertsDir, env.ServerURL)
}
