package main

import (
	"log/slog"
	"path"

	"github.com/hiveot/hivekit/go/cells/transport/clients"
	discoverypkg "github.com/hiveot/hivekit/go/cells/transport/discovery/pkg"

	"github.com/hiveot/hub/bindings/weather"
	weatherpkg "github.com/hiveot/hub/bindings/weather/pkg"
	"github.com/hiveot/hub/lib/plugin"
)

// Start the Weather binding
func main() {
	env := api.NewAppEnvironment("", true)
	cfg := weather.NewWeatherConfig()
	err := env.LoadConfig(&cfg)
	if err != nil {
		slog.Error("Failed loading configuration", "err", err.Error())
	}
	storePath := path.Join(env.StoresDir, env.AppID)

	// locate the server for connecting
	// FIXME: easy way to connect, factory?
	if env.ServerURL == "" {
		cl := discoverypkg.NewDiscoveryClient()
		serverInfo, _ := cl.DiscoverDirectories("", 0, true, nil)
		env.ServerURL = serverInfo[0].AsURL()
	}
	co, err := clients.NewTransportClient("", env.ServerURL, env.CaCert)

	binding := weatherpkg.NewWeatherService(co, storePath, cfg)
	// the plugin filename is the default binding ID
	plugin.StartPlugin(binding, env.AppID, env.CertsDir, env.ServerURL)
}
