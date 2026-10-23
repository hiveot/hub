package runtimepkg

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hub/runtime/internal"
)

// Create a new runtime instance
func NewRuntime(env *api.AppEnvironment) *internal.Runtime {
	rt := internal.NewRuntime(env)
	return rt
}
