package main

import (
	"fmt"
	"os"

	"github.com/hiveot/hivekit/go/api"
	rcdevice_recipe "github.com/hiveot/hivekit/go/factory/recipes/rcdevice"

	"github.com/hiveot/hub/bindings/ipnet/config"
	"github.com/hiveot/hub/bindings/ipnet/service"
)

// Run the ipnet service binding
func main() {

	// use the rcdevice recipe and link the binding ExposedThing
	env := api.NewHiveEnvironment("", true)

	cfg := config.NewIPNetConfig()
	_ = env.LoadConfig("ipnet", &cfg)

	binding := service.NewIpNetBinding(cfg)

	// link it to the RC device recipe that handles connections.
	r, _, err := rcdevice_recipe.NewRCDeviceRecipe(env, binding)
	if err != nil {
		fmt.Println("IPNet startup failed: " + err.Error())
		os.Exit(1)
	}
	r.Run()
}
