package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/hiveot/hivekit/go/api"
	rcdevice_recipe "github.com/hiveot/hivekit/go/factory/recipes/rcdevice"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/owserver"
	owserverpkg "github.com/hiveot/hub/bindings/owserver/pkg"
)

// Start the OWServer protocol binding
func main() {
	env := api.NewAppEnvironment("", true)
	if env.ClientID == "" {
		env.ClientID = env.AppID
	}
	env.Print(1)
	utils.SetLogging(env.LogLevel, "")

	owsConfig := owserver.NewOWServerConfig()
	err := env.LoadConfig(env.ClientID, &owsConfig)
	if err != nil {
		slog.Error("Failed loading configuration", "err", err.Error())
		os.Exit(1)
	}
	storePath := path.Join(env.StoresDir, env.ClientID)
	binding, err := owserverpkg.NewOWServerGateway(env.ClientID, storePath, owsConfig)

	// link it to the RC device recipe that handles connections.
	r, _, err := rcdevice_recipe.NewRCDeviceRecipe(env, binding)
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}
	r.Run()

}
