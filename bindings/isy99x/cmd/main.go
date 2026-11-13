package main

import (
	"fmt"
	"os"

	"github.com/hiveot/hivekit/go/api"
	rcdevice_recipe "github.com/hiveot/hivekit/go/factory/recipes/rcdevice"
	"github.com/hiveot/hub/bindings/isy99x/config"
	"github.com/hiveot/hub/bindings/isy99x/internal"
)

// Start the ISY99x protocol binding
func main() {

	env := api.NewAppEnvironment("", true)
	cfg := config.NewIsy99xConfig()
	_ = env.LoadConfig(env.AppID, &cfg)

	binding, err := internal.NewIsyGateway(env.AppID, cfg)
	if err != nil {
		fmt.Println("NewIsyGateway failed: " + err.Error())
		os.Exit(1)
	}

	// link it to the RC device recipe that handles connections.
	r, _, err := rcdevice_recipe.NewRCDeviceRecipe(env, binding)

	if err != nil {
		fmt.Println("NewRCDeviceRecipe failed: " + err.Error())
		os.Exit(1)
	}
	r.Run()

}
