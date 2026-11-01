package launcher_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hub/services/launcher"
	"github.com/hiveot/hub/services/launcher/service/internal"
)

// Create a new launcher service instance
//
// The sink forwards requests to create service accounts to authn service.
// env is the application environment with bin, certs, log and plugin directories
// cfg is the launcher configuration
func NewLauncherService(
	env *api.HiveEnvironment, cfg *launcher.LauncherConfig) (launcher.ILauncherService, error) {

	svc, err := internal.NewLauncherService(env, cfg)
	return svc, err
}
