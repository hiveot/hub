package service

import (
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/agent"
	"github.com/hiveot/hivekit/go/logging"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/owserver/config"
	"github.com/hiveot/hub/bindings/owserver/service/eds"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/plugin"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	jsoniter "github.com/json-iterator/go"
)

const bindingValuePollIntervalID = "valuePollInterval"
const bindingValuePublishIntervalID = "valueRepublishInterval"
const bindingOWServerAddressID = "owServerAddress"
const bindingMake = "make"

// the key under which custom Thing titles are stored in the state service
const customTitlesKey = "customTitles"

// OWServerBinding is the hub protocol binding plugin for capturing 1-wire OWServer V2 Data
type OWServerBinding struct {
	// Connecting ID and service ID of this binding
	agentID string

	// Configuration of this protocol binding
	config *config.OWServerConfig

	// EDS OWServer client API
	edsAPI *eds.EdsAPI

	// hub client to publish TDs and values and receive actions
	ag *agent.Agent

	// The discovered and publishable things, containing instructions on
	// if and how properties and events are published
	things map[string]*td.TD

	// track the last value for change detection
	// map of [node/device ID] [attribute Title] value
	values map[string]map[string]NodeValueStamp

	// persistent store with device titles
	store buckets.IBucketStore
	// the user edited node names
	customTitles buckets.IBucket

	// nodes by thingID. Used in handling action requests
	nodes map[string]*eds.OneWireNode

	// stop the heartbeat
	stopFn func()

	// lock value updates
	mux sync.RWMutex
}

// CreateBindingTD generates a TD document for this service. Its thingID is the same as its agentID
func (svc *OWServerBinding) CreateBindingTD() *td.TD {
	// This binding exposes the TD of itself.
	// Currently its configuration comes from file.
	tdi := td.NewTD(svc.agentID, "OWServer binding", vocab.ThingService)
	tdi.Description = "Driver for the OWServer V2 Gateway 1-wire interface"

	prop := tdi.AddProperty(bindingMake, "Developer", "", vocab.WoTDataTypeString).
		SetAtType(vocab.PropDeviceMake)

	// these are configured through the configuration file.
	prop = tdi.AddProperty(bindingValuePollIntervalID, "Poll Interval", "Value polling", vocab.WoTDataTypeInteger).
		SetAtType(vocab.PropDevicePollinterval)
	prop.Unit = vocab.UnitSecond

	prop = tdi.AddProperty(bindingValuePublishIntervalID, "Value Republish Interval",
		"Interval the values are published even if they haven't changed.",
		vocab.WoTDataTypeInteger)
	prop.Unit = vocab.UnitSecond

	prop = tdi.AddProperty(bindingOWServerAddressID, "IP Address", "OWServer gateway IP address",
		vocab.WoTDataTypeString).SetAtType(vocab.PropNetAddress)
	return tdi
}

// GetBindingPropValues generates a properties map for attribute and config properties of this binding
func (svc *OWServerBinding) GetBindingPropValues() map[string]any {
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
func (svc *OWServerBinding) Start(ag *agent.Agent) (err error) {
	slog.Info("Starting OWServer binding")
	if svc.config.LogLevel != "" {
		logging.SetLogging(svc.config.LogLevel, "")
	}
	svc.ag = ag
	svc.agentID = ag.GetClientID()
	// Create the adapter for the OWServer 1-wire gateway
	svc.edsAPI = eds.NewEdsAPI(
		svc.config.OWServerURL, svc.config.OWServerLogin, svc.config.OWServerPassword)

	// subscribe to action and configuration requests
	svc.ag.SetRequestHandler(svc.HandleRequest)

	// open the store
	err = svc.store.Open()
	if err != nil {
		slog.Error("Start: Unable to open the state store with custom titles")
	}
	svc.customTitles = svc.store.GetBucket(customTitlesKey)

	// publish this binding's TD document
	tdi := svc.CreateBindingTD()
	svc.things[tdi.ID] = tdi
	tdJSON, _ := jsoniter.MarshalToString(tdi)
	err = digitwin.ThingDirectoryUpdateThing(svc.ag.Consumer, tdJSON)
	//err = svc.ag.UpdateThing(tdi)
	if err != nil {
		slog.Error("failed publishing service TD. Continuing...",
			slog.String("err", err.Error()))
	} else {
		props := svc.GetBindingPropValues()
		err = ag.PubProperties(tdi.ID, props)
	}

	// last, start polling heartbeat
	svc.stopFn = svc.startHeartBeat()
	return nil
}

// heartbeat polls the EDS server every X seconds and publishes TD and value updates
func (svc *OWServerBinding) startHeartBeat() (stopFn func()) {
	slog.Info("Starting heartBeat",
		slog.Int("TD publish interval sec", svc.config.TDInterval),
		slog.Int("polling interval sec", svc.config.PollInterval),
		slog.Int("republish interval sec", svc.config.RepublishInterval),
	)
	var tdCountDown = 0
	var pollCountDown = 0
	var republishCountDown = 0

	stopFn = plugin.StartHeartbeat(time.Second, func() {
		tdCountDown--
		pollCountDown--
		republishCountDown--
		if pollCountDown <= 0 {
			// polling nodes and values takes one call.
			// Since this can take some time, check if client is closed before using it.
			nodes, err := svc.PollNodes()
			svc.mux.RLock()
			isConnected := svc.ag.IsConnected()
			svc.mux.RUnlock()
			if err == nil && isConnected {
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
func (svc *OWServerBinding) Stop() {
	slog.Info("Stopping OWServer binding")

	if svc.stopFn != nil {
		svc.stopFn()
	}
	_ = svc.customTitles.Close()
	_ = svc.store.Close()
	slog.Info("OWServer binding stopped")
}

// NewOWServerBinding creates a new OWServer Protocol Binding service
//
//	config holds the configuration of the service
func NewOWServerBinding(storePath string, config *config.OWServerConfig) *OWServerBinding {

	store := kvbtree.NewKVStore(storePath)

	// these are from hub configuration
	svc := &OWServerBinding{
		config: config,
		store:  store,
		values: make(map[string]map[string]NodeValueStamp),
		nodes:  make(map[string]*eds.OneWireNode),
		things: make(map[string]*td.TD),
	}
	return svc
}
