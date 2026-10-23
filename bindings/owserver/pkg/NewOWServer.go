package owserverpkg

import (
	"path/filepath"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hub/bindings/owserver"
	"github.com/hiveot/hub/bindings/owserver/internal"
)

// NewOWServerBinding creates an instance of the OWServer service
// thingID of the server instance or "" for default
// storePath where to store weather locations data
// cfg service configuration
func NewOWServerBinding(sink api.IHiveCell,
	thingID string, storePath string, cfg *owserver.OWServerConfig) owserver.IOWServer {

	svc := internal.NewOWServerBinding(sink, thingID, storePath, cfg)
	return svc
}

// Factory method for creating a service instance
// todo handle config
func NewOWServerFactory(f api.ICellFactory) (api.IHiveCell, error) {
	env := f.GetEnvironment()
	thingID := owserver.DefaultOWServerThingID
	storePath := filepath.Join(env.StoresDir, thingID)
	cfg := owserver.NewOWServerConfig()
	svc := NewOWServerBinding(nil, thingID, storePath, cfg)
	return svc, nil
}
