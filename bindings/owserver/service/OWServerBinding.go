package service

import (
	"fmt"
	vocab2 "github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/owserver/config"
	"github.com/hiveot/hub/bindings/owserver/service/eds"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"sync"
	"time"
)

const bindingValuePollIntervalID = "valuePollInterval"
const bindingTDIntervalID = "tdPollInterval"
const bindingValuePublishIntervalID = "valueRepublishInterval"
const bindingOWServerAddressID = "owServerAddress"
const bindingMake = "make"

// OWServerBinding is the hub protocol binding plugin for capturing 1-wire OWServer V2 Data
type OWServerBinding struct {
	// Connecting ID and service ID of this binding
	agentID string

	// Configuration of this protocol binding
	config *config.OWServerConfig

	// EDS OWServer client API
	edsAPI *eds.EdsAPI

	// hub client to publish TDs and values and receive actions
	hc hubclient.IHubClient

	// The discovered and publishable things, containing instructions on
	// if and how properties and events are published
	things map[string]*things.TD

	// track the last value for change detection
	// map of [node/device ID] [attribute Title] value
	values map[string]map[string]NodeValueStamp

	// nodes by thingID. Used in handling action requests
	nodes map[string]*eds.OneWireNode

	// stop the heartbeat
	stopFn func()
	// lock value updates
	mux sync.RWMutex
}

// CreateBindingTD generates a TD document for this binding. Its thingID is the same as its agentID
func (svc *OWServerBinding) CreateBindingTD() *things.TD {
	// This binding exposes the TD of itself.
	// Currently its configuration comes from file.
	td := things.NewTD(svc.agentID, "OWServer binding", vocab2.ThingServiceAdapter)
	td.Description = "Driver for the OWServer V2 Gateway 1-wire interface"

	prop := td.AddProperty(bindingMake, vocab2.PropDeviceMake,
		"Developed By", vocab2.WoTDataTypeString)

	// these are configured through the configuration file.
	prop = td.AddProperty(bindingValuePollIntervalID, vocab2.PropDevicePollinterval,
		"Value Polling Interval", vocab2.WoTDataTypeInteger)
	prop.Unit = vocab2.UnitSecond

	prop = td.AddProperty(bindingValuePublishIntervalID, "",
		"Value republish Interval", vocab2.WoTDataTypeInteger)
	prop.Unit = vocab2.UnitSecond

	prop = td.AddProperty(bindingTDIntervalID, vocab2.PropDevicePollinterval,
		"TD Publication Interval", vocab2.WoTDataTypeInteger)
	prop.Unit = vocab2.UnitSecond

	prop = td.AddProperty(bindingOWServerAddressID, vocab2.PropNetAddress,
		"OWServer gateway IP address", vocab2.WoTDataTypeString)
	return td
}

// GetBindingPropValues generates a properties map for attribute and config properties of this binding
func (svc *OWServerBinding) GetBindingPropValues() map[string]string {
	pv := make(map[string]string)
	pv[bindingValuePollIntervalID] = fmt.Sprintf("%d", svc.config.PollInterval)
	pv[bindingTDIntervalID] = fmt.Sprintf("%d", svc.config.TDInterval)
	pv[bindingValuePublishIntervalID] = fmt.Sprintf("%d", svc.config.RepublishInterval)
	pv[bindingOWServerAddressID] = svc.config.OWServerURL
	pv[bindingMake] = "HiveOT"
	return pv
}

// Start the OWServer protocol binding
// This publishes a TD for this binding, starts a background heartbeat.
//
//	hc is the connection with the hubClient to use.
func (svc *OWServerBinding) Start(hc hubclient.IHubClient) (err error) {
	slog.Info("Starting OWServer binding")
	if svc.config.LogLevel != "" {
		logging.SetLogging(svc.config.LogLevel, "")
	}
	svc.hc = hc
	svc.agentID = hc.ClientID()
	// Create the adapter for the OWServer 1-wire gateway
	svc.edsAPI = eds.NewEdsAPI(
		svc.config.OWServerURL, svc.config.OWServerLogin, svc.config.OWServerPassword)

	// subscribe to action and configuration requests
	svc.hc.SetActionHandler(svc.HandleActionRequest)

	// tbd: set the default permissions for managing this binding. is this needed?
	//authzClient := authzclient.NewAuthzClient(hc)
	//err = authzClient.SetPermissions(svc.agentID, svc.agentID,
	//	[]string{
	//		api.ClientRoleManager,
	//		api.ClientRoleAdmin,
	//		api.ClientRoleService})

	// publish this binding's TD document
	td := svc.CreateBindingTD()
	svc.things[td.ID] = td
	err = svc.hc.PubTD(td)
	if err != nil {
		slog.Error("failed publishing service TD. Continuing...",
			slog.String("err", err.Error()))
	} else {
		props := svc.GetBindingPropValues()
		_ = svc.hc.PubProps(td.ID, props)
	}

	// last, start polling heartbeat
	svc.stopFn = svc.startHeartBeat()
	return nil
}

// heartbeat polls the EDS server every X seconds and publishes TD and value updates
func (svc *OWServerBinding) startHeartBeat() (stopFn func()) {
	slog.Info("Starting heartBeat", "TD publish interval", svc.config.TDInterval, "polling", svc.config.PollInterval)
	var tdCountDown = 0
	var pollCountDown = 0
	var republishCountDown = svc.config.RepublishInterval

	stopFn = plugin.StartHeartbeat(time.Second, func() {
		tdCountDown--
		pollCountDown--
		republishCountDown--
		if pollCountDown <= 0 {
			// polling nodes and values takes one call
			nodes, err := svc.PollNodes()
			if err == nil {
				if tdCountDown <= 0 {
					// Every TDInterval publish the full TD's
					err = svc.PublishNodeTDs(nodes)
					tdCountDown = svc.config.TDInterval
				}
				// publish changed values or periodically for publishing all values
				forceRepublish := pollCountDown < 0
				if pollCountDown <= 0 {
					pollCountDown = svc.config.RepublishInterval
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
		svc.stopFn = nil
	}
}

// NewOWServerBinding creates a new OWServer Protocol Binding service
//
//	config holds the configuration of the service
func NewOWServerBinding(config *config.OWServerConfig) *OWServerBinding {

	// these are from hub configuration
	svc := &OWServerBinding{
		config: config,
		values: make(map[string]map[string]NodeValueStamp),
		nodes:  make(map[string]*eds.OneWireNode),
		things: make(map[string]*things.TD),
	}
	return svc
}
