package cmd

import (
	"github.com/hiveot/hub/bindings/weather/config"
	"github.com/hiveot/hub/bindings/weather/service"
	"github.com/hiveot/hub/lib/plugin"
	"path"
)

// Start the Weather binding
func main() {
	env := plugin.GetAppEnvironment("", true)
	cfg := config.NewWeatherConfig()
	_ = env.LoadConfig(&cfg)
	storePath := path.Join(env.StoresDir, env.ClientID)

	binding := service.NewWeatherBinding(storePath, cfg)
	// the plugin filename is the default binding ID
	plugin.StartPlugin(binding, env.ClientID, env.CertsDir, env.ServerURL)
}
