package launcherpkg

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hub/services/launcher"
	"github.com/hiveot/hub/services/launcher/internal"
)

// Create a new launcher service instance
//
// The sink forwards requests to create service accounts to authn service.
// env is the application environment with bin, certs, log and plugin directories
// cfg is the launcher configuration
func NewLauncherService(
	sink api.IHiveCell, env *api.AppEnvironment, cfg *launcher.LauncherConfig) launcher.ILauncherService {
	svc := internal.NewLauncherService(sink, env, cfg)
	return svc
}
