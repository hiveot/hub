package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/hiveot/hivekit/go/api"
	rcdevice_recipe "github.com/hiveot/hivekit/go/factory/recipes/rcdevice"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/isy99x/config"
	"github.com/hiveot/hub/bindings/isy99x/internal"
)

// Start the ISY99x protocol binding
func main() {

	env := api.NewAppEnvironment("", true)
	if env.ClientID == "" {
		env.ClientID = env.AppID
	}
	env.Print(1)
	utils.SetLogging(env.LogLevel, "")

	cfg := config.NewIsy99xConfig()
	err := env.LoadConfig(env.ClientID, &cfg)
	if err != nil {
		slog.Error("Failed loading configuration", "err", err.Error())
		os.Exit(1)
	}

	binding, err := internal.NewIsyGateway("isy99x", cfg)
	if err != nil {
		fmt.Println("NewIsyGateway failed: " + err.Error())
		os.Exit(1)
	}

	// link it to the RC device recipe that handles connections.
	r, _, err := rcdevice_recipe.NewRCDeviceRecipe(env, binding)

	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}
	r.Run()

}
