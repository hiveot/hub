package internal

import (
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
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

// OWServer is the hub protocol binding plugin for capturing 1-wire OWServer V2 Data
type OWServer struct {
	*thing.ExposedThing

	// Connecting ID and service ID of this binding
	// serviceID string

	// Configuration of this protocol binding
	config *owserver.OWServerConfig

	// EDS OWServer client API
	edsAPI *eds.EdsAPI

	// The discovered and publishable things, containing instructions on
	// if and how properties and events are published
	things map[string]*td.TD

	// track the last value for change detection
	// map of [node/device ID] [attribute Title] value
	values map[string]map[string]NodeValueStamp

	// persistent store with device titles
	store bucketstore.IBucketStore
	// the user edited node names
	customTitles bucketstore.IBucket

	// nodes by thingID. Used in handling action requests
	nodes map[string]*eds.OneWireNode

	// stop the heartbeat
	stopFn func()

	// lock value updates
	mux sync.RWMutex
}

// CreateBindingTD generates a TD document for this service. Its thingID is the same as its agentID
func (svc *OWServer) CreateBindingTD() *td.TD {
	// This binding exposes the TD of itself.
	// Currently its configuration comes from file.
	tdi := td.NewTD(svc.GetID(), "OWServer binding", vocab.DeviceTypeService)
	tdi.Description = "Driver for the OWServer V2 Gateway 1-wire interface"

	prop := tdi.AddProperty(bindingMake, "Developer", "", td.DataTypeString).
		SetAtType(vocab.PropDeviceMake)

	// these are configured through the configuration file.
	prop = tdi.AddProperty(bindingValuePollIntervalID, "Poll Interval", "Value polling", td.DataTypeInteger).
		SetAtType(vocab.PropDevicePollinterval)
	prop.Unit = vocab.UnitSecond

	prop = tdi.AddProperty(bindingValuePublishIntervalID, "Value Republish Interval",
		"Interval the values are published even if they haven't changed.",
		td.DataTypeInteger)
	prop.Unit = vocab.UnitSecond

	prop = tdi.AddProperty(bindingOWServerAddressID, "IP Address", "OWServer gateway IP address",
		td.DataTypeString).SetAtType(vocab.PropNetAddress)
	return tdi
}

// GetBindingPropValues generates a properties map for attribute and config properties of this binding
func (svc *OWServer) GetBindingPropValues() map[string]any {
	pv := make(map[string]any)
	pv[bindingValuePollIntervalID] = svc.config.PollInterval
	pv[bindingValuePublishIntervalID] = svc.config.RepublishInterval
	pv[bindingOWServerAddressID] = svc.config.OWServerURL
	pv[bindingMake] = "HiveOT"
	return pv
}

// Start the OWServer protocol binding
// This publishes a TD for this binding, starts a background heartbeat.
//
//	ag is the agent connection for receiving requests and sending responses.
func (svc *OWServer) Start() {
	slog.Info("Starting OWServer binding")
	if svc.config.LogLevel != "" {
		utils.SetLogging(svc.config.LogLevel, "")
	}
	// Create the adapter for the OWServer 1-wire gateway
	svc.edsAPI = eds.NewEdsAPI(
		svc.config.OWServerURL, svc.config.OWServerLogin, svc.config.OWServerPassword)

	svc.customTitles = svc.store.GetBucket(customTitlesKey)

	// publish this binding's properties including its TD
	tdoc := svc.CreateBindingTD()
	svc.things[tdoc.ID] = tdoc

	svc.WriteTD(tdoc)
	props := svc.GetBindingPropValues()
	svc.PubProperties(tdoc.ID, props, true)

	// last, start polling heartbeat
	svc.stopFn = svc.startHeartBeat()
}

// heartbeat polls the EDS server every X seconds and publishes TD and value updates
func (svc *OWServer) startHeartBeat() (stopFn func()) {
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
			nodes, err := svc.PollNodes()
			if err == nil {
				if tdCountDown <= 0 {
					// Every TDInterval publish the full TD's
					err = svc.PublishNodeTDs(nodes)
					tdCountDown = svc.config.TDInterval
				}
				// publish changed values or periodically for publishing all values
				forceRepublish := false
				if republishCountDown <= 0 {
					republishCountDown = svc.config.RepublishInterval
					forceRepublish = true
				}
				err = svc.PublishNodeValues(nodes, forceRepublish)
			}
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
func (svc *OWServer) Stop() {
	slog.Info("Stopping OWServer binding")

	if svc.stopFn != nil {
		svc.stopFn()
	}
	_ = svc.customTitles.Close()
	_ = svc.store.Close()
	slog.Info("OWServer binding stopped")
}

// NewOWServerService creates a new OWServer protocol binding.
//
// This returns a HiveKit cell that can be linked to a HiveKit transport or included
// as part of a hivekit recipe.
//
// For testing this support the use of a simulation file instead of a OWServer URL.
//
//	serviceID is the instance ThingID. Typically the environment appID.
//	 Use "" for default if only a single instance is used.
//	storePath is the location to store custom Thing titles. (OWServer doesnt store this)
//	 Use "" for in-memory storage.
//	config holds the configuration of the service
func NewOWServerBinding(serviceID string, storePath string, config *owserver.OWServerConfig) (*OWServer, error) {

	if serviceID == "" {
		serviceID = owserver.DefaultOWServerThingID
	}

	store, err := kvbtreestore.OpenKVBTreeStore(storePath)
	if err != nil {
		return nil, err
	}

	// these are from hub configuration
	svc := &OWServer{
		ExposedThing: thing.NewExposedThing(serviceID, nil),
		config:       config,
		store:        store,
		values:       make(map[string]map[string]NodeValueStamp),
		nodes:        make(map[string]*eds.OneWireNode),
		things:       make(map[string]*td.TD),
	}

	var _ owserver.IOWServer = svc // interface check
	return svc, nil
}
