package main

import (
	"github.com/hiveot/hub/bindings/weather/config"
	"github.com/hiveot/hub/bindings/weather/service"
	"github.com/hiveot/hub/lib/plugin"
	"log/slog"
	"path"
)

// Start the Weather binding
func main() {
	env := plugin.GetAppEnvironment("", true)
	cfg := config.NewWeatherConfig()
	err := env.LoadConfig(&cfg)
	if err != nil {
		slog.Error("Failed loading configuration", "err", err.Error())
	}
	storePath := path.Join(env.StoresDir, env.ClientID)

	binding := service.NewWeatherBinding(storePath, cfg)
	// the plugin filename is the default binding ID
	plugin.StartPlugin(binding, env.ClientID, env.CertsDir, env.ServerURL)
}
