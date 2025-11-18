package main

import (
	"github.com/hiveot/hivehub/bindings/ipnet/config"
	"github.com/hiveot/hivehub/bindings/ipnet/service"
	"github.com/hiveot/hivehub/lib/plugin"
)

// Run the ipnet service binding
func main() {
	env := plugin.GetAppEnvironment("", true)
	cfg := config.NewIPNetConfig()
	_ = env.LoadConfig(&cfg)
	svc := service.NewIpNetBinding(cfg)

	plugin.StartPlugin(svc, env.ClientID, env.CertsDir, env.ServerURL)
}
