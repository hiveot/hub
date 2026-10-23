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

	env := api.NewHiveEnvironment("", true)
	cfg := config.NewIsy99xConfig()
	_ = env.LoadConfig(env.AppID, &cfg)

	binding := internal.NewIsyBinding(env.AppID, cfg)

	// link it to the RC device recipe that handles connections.
	r, _, err := rcdevice_recipe.NewRCDeviceRecipe(env, binding)
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}
	r.Run()

}
