package internal

import (
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/bucketstore"
	"github.com/hiveot/hivekit/go/cells/bucketstore/kvbtreestore"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/owserver"
	"github.com/hiveot/hub/bindings/owserver/internal/eds"
)

const bindingValuePollIntervalID = "valuePollInterval"
const bindingValuePublishIntervalID = "valueRepublishInterval"
const bindingOWServerAddressID = "owServerAddress"
const bindingMake = "make"

// the key under which custom Thing titles are stored in the state service
const customTitlesKey = "customTitles"

// OWServerGateway is the 1-wire gateway for capturing 1-wire OWServerGateway V2 Data
type OWServerGateway struct {
	*thing.ExposedThing

	// Connecting ID and service ID of this binding
	// serviceID string

	// Configuration of this protocol binding
	config *owserver.OWServerConfig

	// EDS OWServer client API
	edsAPI *eds.EdsAPI

	// The 1-wire Things discovered via the gateway
	owThings map[string]*StandardOWThing

	// The discovered and publishable things, containing instructions on
	// if and how properties and events are published
	// things map[string]*wot.TD

	// track the last value for change detection
	// map of [node/device ID] [attribute Title] value
	// values map[string]map[string]NodeValueStamp

	// persistent store with device titles
	store bucketstore.IBucketStore
	// the user edited node names
	customTitles bucketstore.IBucket

	// nodes by thingID. Used in handling action requests
	// nodes map[string]*eds.OneWireNode

	// stop the heartbeat
	stopFn func()

	// lock value updates
	mux sync.RWMutex
}

// CreateBindingTD generates a TD document for this service. Its thingID is the same as its agentID
func (svc *OWServerGateway) CreateBindingTD() *wot.TD {
	// This binding exposes the TD of itself.
	// Currently its configuration comes from file.
	tdi := wot.NewTD(svc.GetID(), "OWServer binding", vocab.DeviceTypeService)
	tdi.Description = "Driver for the OWServer V2 Gateway 1-wire interface"

	prop := tdi.AddProperty(bindingMake, "Developer", "", wot.DataTypeString).
		SetAtType(vocab.PropDeviceMake)

	// these are configured through the configuration file.
	prop = tdi.AddProperty(bindingValuePollIntervalID, "Poll Interval", "Value polling", wot.DataTypeInteger).
		SetAtType(vocab.PropDevicePollinterval)
	prop.Unit = vocab.UnitSecond

	prop = tdi.AddProperty(bindingValuePublishIntervalID, "Value Republish Interval",
		"Interval the values are published even if they haven't changed.",
		wot.DataTypeInteger)
	prop.Unit = vocab.UnitSecond

	prop = tdi.AddProperty(bindingOWServerAddressID, "IP Address", "OWServer gateway IP address",
		wot.DataTypeString).SetAtType(vocab.PropNetAddress)
	return tdi
}

// GetBindingPropValues generates a properties map for attribute and config properties of this binding
func (svc *OWServerGateway) GetBindingPropValues() map[string]any {
	pv := make(map[string]any)
	pv[bindingValuePollIntervalID] = svc.config.PollInterval
	pv[bindingValuePublishIntervalID] = svc.config.RepublishInterval
	pv[bindingOWServerAddressID] = svc.config.OWServerURL
	pv[bindingMake] = "HiveOT"
	return pv
}

func (svc *OWServerGateway) GetOWThing(thingID string) (*StandardOWThing, bool) {
	svc.mux.RLock()
	defer svc.mux.RUnlock()
	owt, found := svc.owThings[thingID]
	return owt, found
}

// Start the OWServer protocol binding
// This publishes a TD for this binding, starts a background heartbeat.
//
//	ag is the agent connection for receiving requests and sending responses.
func (svc *OWServerGateway) Start() {

	// publish this binding's properties including its TD
	tdoc := svc.CreateBindingTD()
	tdoc.ID = svc.GetID()
	tdoc.SetType(owserver.OWServerCellType)
	svc.WriteTD(tdoc)

	props := svc.GetBindingPropValues()
	svc.PubProperties(props, true)

	// poll the nodes and publish their things too
	svc.UpdateThings(true)
	svc.UpdateThingValues(true)

	// last, start polling heartbeat
	svc.stopFn = svc.startHeartBeat()
}

// heartbeat polls the EDS server every X seconds and publishes TD and value updates
func (svc *OWServerGateway) startHeartBeat() (stopFn func()) {
	slog.Info("Starting heartBeat",
		slog.Int("TD publish interval sec", svc.config.TDInterval),
		slog.Int("polling interval sec", svc.config.PollInterval),
		slog.Int("republish interval sec", svc.config.RepublishInterval),
	)
	var tdCountDown = 0
	var pollCountDown = 0
	var republishCountDown = 0

	stopFn = utils.StartHeartbeat(time.Second, func() {
		tdCountDown--
		pollCountDown--
		republishCountDown--
		if pollCountDown <= 0 {

			// polling nodes and values takes one call.
			// Since this can take some time, check if client is closed before using it.
			if tdCountDown <= 0 {
				// Every TDInterval publish the full TD's
				svc.UpdateThings(true)
				tdCountDown = svc.config.TDInterval
			}
			// publish changed values or periodically for publishing all values
			onlyChanges := true
			if republishCountDown <= 0 {
				republishCountDown = svc.config.RepublishInterval
				onlyChanges = false
			}
			err := svc.UpdateThingValues(onlyChanges)

			pollCountDown = svc.config.PollInterval
			// slow down if polling fails
			if err != nil {
				pollCountDown = svc.config.PollInterval * 5
			}
		}
	})
	return stopFn
}

// Stop the heartbeat and remove subscriptions
// This does not close the given hubclient connection.
func (svc *OWServerGateway) Stop() {
	slog.Info("Stopping OWServer binding")

	if svc.stopFn != nil {
		svc.stopFn()
	}
	_ = svc.customTitles.Close()
	_ = svc.store.Close()
	slog.Info("OWServer binding stopped")
}

// UpdateThingValues polls the OWServer hub for changed Thing values.
// Set 'onlyChanges' to only publish changed values.
func (svc *OWServerGateway) UpdateThingValues(onlyChanges bool) error {

	nodes, err := svc.edsAPI.PollNodes()
	if err == nil {
		for _, node := range nodes {
			thingID := node.ROMId
			owt, found := svc.owThings[thingID]
			if found {
				title, err := svc.customTitles.Get(thingID)
				titleStr := ""
				if err == nil && title != nil {
					titleStr = string(title)
				}
				owt.UpdateProperties(node, titleStr, onlyChanges)
			}
		}
	}
	return err
}

// NewOWServerGateway creates a new OWServer gateway devices that servers
// multiple 1-wire devices.
//
// This returns a HiveKit cell that can be linked to a HiveKit transport or included
// as part of a hivekit recipe.
//
// For testing this support the use of a simulation file instead of a OWServer URL.
//
//	serviceID is the instance ThingID. Typically the app clientID.
//	 Use "" for the cell type default if only a single instance is used.
//	storePath is the location to store custom Thing titles. (OWServer doesnt store this)
//	 Use "" for in-memory storage.
//	config holds the configuration of the service
func NewOWServerGateway(
	serviceID string, storePath string, config *owserver.OWServerConfig) (*OWServerGateway, error) {

	if serviceID == "" {
		serviceID = owserver.OWServerCellType
	}
	slog.Info("New OWServer binding")

	store, err := kvbtreestore.OpenKVBTreeStore(storePath)
	if err != nil {
		return nil, err
	}

	// Create the adapter for the OWServer 1-wire gateway
	edsAPI := eds.NewEdsAPI(
		config.OWServerURL, config.OWServerLogin, config.OWServerPassword)

	// these are from hub configuration
	svc := &OWServerGateway{
		ExposedThing: thing.NewExposedThing(serviceID, nil),
		config:       config,
		customTitles: store.GetBucket(customTitlesKey),
		store:        store,
		owThings:     make(map[string]*StandardOWThing),
		// values:       make(map[string]map[string]NodeValueStamp),
		// nodes:        make(map[string]*eds.OneWireNode),
		// things:       make(map[string]*wot.TD),
		edsAPI: edsAPI,
	}

	var _ owserver.IOWServer = svc // interface check
	return svc, nil
}
