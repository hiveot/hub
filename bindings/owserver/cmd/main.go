package main

import (
	"os"

	"github.com/hiveot/hivekit/go/api"
	factorypkg "github.com/hiveot/hivekit/go/cells/factory/pkg"
	"github.com/hiveot/hub/bindings/owserver"
	owserverpkg "github.com/hiveot/hub/bindings/owserver/pkg"
)

// Start the OWServer protocol binding
func main() {
	env := api.NewAppEnvironment("", true)
	f := factorypkg.NewModuleFactory(env, nil)

	// Use the recipe template for building devices using reverse connection.
	// the factory is configured for auto-discovery the hub and authentication
	// using the app environment.
	//
	// Recipe contains discovery, logging, client auth (token|cert), and transport client
	// The app module is added at the end.
	owserverModDef := api.CellDefinition{
		Type:        owserver.DefaultOWServerThingID,
		Constructor: owserverpkg.NewOWServerFactory,
	}
	r := factorypkg.NewRCDeviceRecipe(f, &owserverModDef)

	// start the recipe services
	err := r.Start()
	if err != nil {
		os.Exit(1)
	}
}
