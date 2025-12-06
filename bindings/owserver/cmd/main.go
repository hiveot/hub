package main

import (
	"path"

	"github.com/hiveot/hivekit/go/lib/plugin"
	"github.com/hiveot/hub/bindings/owserver/config"
	"github.com/hiveot/hub/bindings/owserver/service"
)

// Start the OWServer protocol binding
func main() {
	env := plugin.GetAppEnvironment("", true)
	cfg := config.NewConfig()
	_ = env.LoadConfig(&cfg)
	storePath := path.Join(env.StoresDir, env.ClientID)
	binding := service.NewOWServerBinding(storePath, cfg)
	plugin.StartPlugin(binding, env.ClientID, env.CertsDir, env.ServerURL)
}
