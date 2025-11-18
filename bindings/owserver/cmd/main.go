package main

import (
	"github.com/hiveot/hivehub/bindings/owserver/config"
	"github.com/hiveot/hivehub/bindings/owserver/service"
	"github.com/hiveot/hivehub/lib/plugin"
	"path"
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
