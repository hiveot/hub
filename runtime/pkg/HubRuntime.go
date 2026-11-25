package pkg

import (
	"github.com/hiveot/hivekit/go/api"
	gatewayrecipe "github.com/hiveot/hivekit/go/factory/recipes/gateway"
)

type HubRuntime struct {
	*gatewayrecipe.GatewayRecipe
	f api.ICellFactory
	// r api.IHiveCell
}

// func (rt *HubRuntime) Start() {
// 	rt.r.Start()

// }
// func (rt *HubRuntime) Stop() {
// 	rt.r.Stop()
// }

// Create a new hub runtime instance.
// This runs the gateway recipe.
// Call Start() before use.
func NewHubRuntime(env *api.AppEnvironment) (*HubRuntime, api.ICellFactory, error) {
	// this recipe does most of the work
	r, f, err := gatewayrecipe.NewGatewayRecipe(env)

	rt := &HubRuntime{
		GatewayRecipe: r,
		// r: r,
		f: f,
	}
	return rt, f, err
}
