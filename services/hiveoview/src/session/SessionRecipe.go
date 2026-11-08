package session

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/cells/directory"
	directory_client "github.com/hiveot/hivekit/go/cells/directory/client"
	"github.com/hiveot/hivekit/go/cells/history"
	history_client "github.com/hiveot/hivekit/go/cells/history/client"
	"github.com/hiveot/hivekit/go/cells/transport/wss"
	wss_client "github.com/hiveot/hivekit/go/cells/transport/wss/client"
	"github.com/hiveot/hivekit/go/cells/vcache"
	vcache_service "github.com/hiveot/hivekit/go/cells/vcache/service"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
)

// SessionRecipe is a HiveKit recipe for use by web client sessions.
// It holds a client connection for the hub, a directory for obtaining Thing TDs,
// and a value cache with the last known values of Things.

var SessionRecipeChain = []api.CellDefinition{
	{
		// use a history client to read historical values
		Type:        history.ReadHistoryClientCellType,
		Constructor: history_client.NewReadHistoryClientFactory,
	},
	{
		// use a value cache to track affordance values
		Type:        vcache.ValueCacheCellType,
		Constructor: vcache_service.NewValueCacheServiceFactory,
	},
	{
		// use a directory client to read thing TDs
		Type:        directory.DirectoryClientCellType,
		Constructor: directory_client.NewDirectoryClientFactory,
	},
	// TBD, no reconnect for now
	{
		// use a websocket client to connect to the hub
		Type:        wss.WotWebsocketClientCellType,
		Constructor: wss_client.NewWotWssClientFactory,
	}}

// This is designed for easy communication with the hub by the session:
// 1. Read the directory
// 2. Read Thing property and event values
// 3. Be notificied of value updates
// 4. Invoke actions
type SessionRecipe struct {
	*consumer.Consumer
	f         api.ICellFactory
	formation api.IHiveCell
}

// Return the directory client from this recipe
func (r *SessionRecipe) GetDirectory() directory.IDirectoryClient {
	dirClient := api.GetFactoryCell[directory.IDirectoryClient](r.f, directory.DirectoryClientCellType)
	return dirClient
}

// Return the history client from this recipe
func (r *SessionRecipe) GetHistoryClient() *history_client.ReadHistoryClient {
	histClient := api.GetFactoryCell[*history_client.ReadHistoryClient](r.f, history.ReadHistoryClientCellType)
	return histClient
}

// Return the directory client from this recipe
func (r *SessionRecipe) GetValues() vcache.IValueCacheService {
	vcache := api.GetFactoryCell[vcache.IValueCacheService](r.f, vcache.ValueCacheCellType)
	return vcache
}

// Return the client connection from this recipe
func (r *SessionRecipe) GetConnection() api.ITransportClient {
	vcache := api.GetFactoryCell[api.ITransportClient](r.f, wss.WotWebsocketClientCellType)
	return vcache
}

// Return the connection status
func (r *SessionRecipe) IsConnected() bool {
	cc := r.GetConnection()
	status := cc.GetConnectionStatus()
	return status == api.StatusConnected
}

// Start the recipe and factory.
func (r *SessionRecipe) Start() {
	r.f.Start()
}

// Stop the recipe and factory.
func (r *SessionRecipe) Stop() {
	r.f.Stop()
}

// NewSessionRecipe returns a ready to use consumer of the hub for use by
// the hiveoview web client session.
//
// The resulting consumer is the start of the recipe chain.
// Invoke Start on the recipe to run the consumer.
func NewSessionRecipe(env *api.HiveEnvironment) (r *SessionRecipe, err error) {

	f := factory_service.NewCellFactory(env, nil)
	formation, err := factory_service.NewChainFormation(f, SessionRecipeChain, nil, false)

	co := consumer.NewConsumer(nil, nil)
	r = &SessionRecipe{
		Consumer:  co,
		f:         f,
		formation: formation,
	}
	// link this session to the chain to send requests and receive notifications
	formation.SetNotificationSink(r)
	r.SetRequestSink(formation)
	return r, err
}
