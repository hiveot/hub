package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/hiveot/hivekit/go/api"
	rcdevice_recipe "github.com/hiveot/hivekit/go/factory/recipes/rcdevice"
	"github.com/hiveot/hub/bindings/weather"
	weather_pkg "github.com/hiveot/hub/bindings/weather/pkg"
)

// Start the Weather binding
func main() {
	env := api.NewHiveEnvironment("", true)

	cfg := weather.NewWeatherConfig()
	err := env.LoadConfig(env.AppID, &cfg)
	if err != nil {
		slog.Error("Failed loading configuration", "err", err.Error())
	}
	storePath := path.Join(env.StoresDir, env.AppID)
	binding, err := weather_pkg.NewWeatherService("", storePath, cfg)

	// link it to the RC device recipe that handles connections.
	r, _, err := rcdevice_recipe.NewRCDeviceRecipe(env, binding)
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}
	r.Run()
}
