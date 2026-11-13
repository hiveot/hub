package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/hiveot/hivekit/go/api"
	rcdevice_recipe "github.com/hiveot/hivekit/go/factory/recipes/rcdevice"
	"github.com/hiveot/hub/bindings/owserver"
	owserverpkg "github.com/hiveot/hub/bindings/owserver/pkg"
)

// Start the OWServer protocol binding
func main() {
	env := api.NewAppEnvironment("", true)

	owsConfig := owserver.NewOWServerConfig()
	err := env.LoadConfig(env.AppID, &owsConfig)
	if err != nil {
		slog.Error("Failed loading configuration", "err", err.Error())
	}
	storePath := path.Join(env.StoresDir, env.ClientID)
	binding, err := owserverpkg.NewOWServerBinding("", storePath, owsConfig)

	// link it to the RC device recipe that handles connections.
	r, _, err := rcdevice_recipe.NewRCDeviceRecipe(env, binding)
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}
	r.Run()

}
