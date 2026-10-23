package owserverpkg

import (
	"path/filepath"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hub/bindings/owserver"
	"github.com/hiveot/hub/bindings/owserver/internal"
)

// NewOWServerBinding creates an instance of the OWServer service
//
//	thingID of the server instance or "" for default
//	storePath is the location to store custom Thing titles. (OWServer doesnt store this)
//	cfg service configuration
func NewOWServerBinding(
	thingID string, storePath string, cfg *owserver.OWServerConfig) (owserver.IOWServer, error) {

	svc, err := internal.NewOWServerBinding(thingID, storePath, cfg)
	return svc, err
}

// Factory method for creating a service instance
// todo handle config
func NewOWServerFactory(f api.ICellFactory) (api.IHiveCell, error) {
	env := f.GetEnvironment()
	thingID := owserver.DefaultOWServerThingID
	storePath := filepath.Join(env.StoresDir, thingID)
	cfg := owserver.NewOWServerConfig()
	svc, err := NewOWServerBinding(thingID, storePath, cfg)
	return svc, err
}
