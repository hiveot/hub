package main

import (
	"github.com/hiveot/hub/bindings/owserver/config"
	"github.com/hiveot/hub/bindings/owserver/service"
	"github.com/hiveot/hub/lib/plugin"
)

// Start the OWServer protocol binding
func main() {
	env := plugin.GetAppEnvironment("", true)
	cfg := config.NewConfig()
	_ = env.LoadConfig(&cfg)
	binding := service.NewOWServerBinding(cfg)
	plugin.StartPlugin(binding, env.ClientID, env.CertsDir)
}
