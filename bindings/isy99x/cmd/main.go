package main

import (
	"github.com/hiveot/hivehub/bindings/isy99x/config"
	"github.com/hiveot/hivehub/bindings/isy99x/service"
	"github.com/hiveot/hivehub/lib/plugin"
)

// Start the ISY99x protocol binding
func main() {

	env := plugin.GetAppEnvironment("", true)
	cfg := config.NewIsy99xConfig()
	_ = env.LoadConfig(&cfg)
	binding := service.NewIsyBinding(cfg)
	plugin.StartPlugin(binding, env.ClientID, env.CertsDir, env.ServerURL)
}
