package main

import (
	"github.com/hiveot/hub/bindings/isy99x/config"
	"github.com/hiveot/hub/bindings/isy99x/service"
	"github.com/hiveot/hub/lib/plugin"
)

// Start the ISY99x protocol binding
func main() {

	env := plugin.GetAppEnvironment("", true)
	cfg := config.NewIsy99xConfig()
	_ = env.LoadConfig(&cfg)
	binding := service.NewIsyBinding(cfg)
	plugin.StartPlugin(binding, env.ClientID, env.CertsDir)
}
