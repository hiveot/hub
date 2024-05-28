package main

import (
	"github.com/hiveot/hub/bindings/ipnet/config"
	"github.com/hiveot/hub/bindings/ipnet/service"
	"github.com/hiveot/hub/lib/plugin"
)

// Run the ipnet service binding
func main() {
	env := plugin.GetAppEnvironment("", true)
	cfg := config.NewIPNetConfig()
	_ = env.LoadConfig(&cfg)
	svc := service.NewIpNetBinding(cfg)

	plugin.StartPlugin(svc, env.ClientID, env.CertsDir)
}
