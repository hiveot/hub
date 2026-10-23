package pkg

import (
	"github.com/hiveot/hivekit/go/api"
	gatewayrecipe "github.com/hiveot/hivekit/go/factory/recipes/gateway"
)

type HubRuntime struct {
	f api.IFactory
	r *api.IHiveCell
}

func (rt *HubRuntime) Start() {
	rt.r.Start()

}
func (rt *HubRuntime) Stop() {
	rt.r.Stop()
}

func NewHubRuntime(env *api.HiveEnvironment) (*HubRuntime, error) {

	r, f, err := gatewayrecipe.NewGatewayRecipe(env)

	rt := &HubRuntime{
		r: r,
		f: f,
	}
	return rt, err
}
