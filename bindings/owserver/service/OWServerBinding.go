package service

import (
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
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

// OWServerBinding is the hub protocol binding plugin for capturing 1-wire OWServer V2 Data
type OWServerBinding struct {
	// Configuration of this protocol binding
	config *config.OWServerConfig

	// EDS OWServer client API
	edsAPI *eds.EdsAPI

	// hub client to publish TDs and values and receive actions
	hc *hubclient.HubClient

	// track the last value for change detection
	// map of [node/device ID] [attribute Title] value
	values map[string]map[string]NodeValueStamp

	// nodes by thingID. Used in handling action requests
	nodes map[string]*eds.OneWireNode

	// Map of previous node values [nodeID][attrName]value
	// nodeValues map[string]map[string]string

	// stop the heartbeat
	stopFn func()
	mu     sync.Mutex
}

// CreateBindingTD generates a TD document for this binding
func (svc *OWServerBinding) CreateBindingTD() *things.TD {
	thingID := svc.hc.ClientID()
	td := things.NewTD(thingID, "OWServer binding", vocab.ThingServiceAdapter)
	// these are configured through the configuration file.
	prop := td.AddProperty(bindingValuePollIntervalID, vocab.PropDevicePollinterval,
		"Value Polling Interval", vocab.WoTDataTypeInteger)
	prop.Unit = vocab.UnitSecond

	prop = td.AddProperty(bindingValuePublishIntervalID, "",
		"Value republish Interval", vocab.WoTDataTypeInteger)
	prop.Unit = vocab.UnitSecond

	prop = td.AddProperty(bindingTDIntervalID, vocab.PropDevicePollinterval,
		"TD Publication Interval", vocab.WoTDataTypeInteger)
	prop.Unit = vocab.UnitSecond

	prop = td.AddProperty(bindingOWServerAddressID, vocab.PropNetAddress,
		"OWServer gateway IP address", vocab.WoTDataTypeString)
	return td
}

// MakeBindingProps generates a properties map for attribute and config properties of this binding
func (svc *OWServerBinding) MakeBindingProps() map[string]string {
	pv := make(map[string]string)
	pv[bindingValuePollIntervalID] = fmt.Sprintf("%d", svc.config.PollInterval)
	pv[bindingTDIntervalID] = fmt.Sprintf("%d", svc.config.TDInterval)
	pv[bindingValuePublishIntervalID] = fmt.Sprintf("%d", svc.config.RepublishInterval)
	pv[bindingOWServerAddressID] = svc.config.OWServerURL
	return pv
}

// Start the OWServer protocol binding
// This publishes a TD for this binding, starts a background heartbeat.
//
//	hc is the connection with the hubClient to use.
func (svc *OWServerBinding) Start(hc *hubclient.HubClient) (err error) {
	slog.Warn("Starting OWServer binding")
	if svc.config.LogLevel != "" {
		logging.SetLogging(svc.config.LogLevel, "")
	}
	svc.hc = hc
	// Create the adapter for the OWServer 1-wire gateway
	svc.edsAPI = eds.NewEdsAPI(
		svc.config.OWServerURL, svc.config.OWServerLogin, svc.config.OWServerPassword)

	// TODO: restore svc configuration

	// subscribe to action and configuration requests
	svc.hc.SetActionHandler(svc.HandleActionRequest)
	svc.hc.SetConfigHandler(svc.HandleConfigRequest)

	//myProfile := authclient.NewProfileClient(svc.hc)
	//err = myProfile.SetServicePermissions(WriteConfigCap, []string{
	//	authapi.ClientRoleManager,
	//	authapi.ClientRoleAdmin,
	//	authapi.ClientRoleService})

	// publish this binding's TD document
	td := svc.CreateBindingTD()
	err = svc.hc.PubTD(td)
	if err != nil {
		slog.Error("failed publishing service TD. Continuing...",
			slog.String("err", err.Error()))
	} else {
		props := svc.MakeBindingProps()
		_ = svc.hc.PubProps(td.ID, props)
	}

	// last, start polling heartbeat
	svc.stopFn = svc.startHeartBeat()
	return nil
}

// heartbeat polls the EDS server every X seconds and publishes updates
func (svc *OWServerBinding) startHeartBeat() (stopFn func()) {
	slog.Info("Starting heartBeat", "TD publish interval", svc.config.TDInterval, "polling", svc.config.PollInterval)
	var tdCountDown = 0
	var pollCountDown = 0

	stopFn = plugin.StartHeartbeat(time.Second, func() {
		tdCountDown--
		pollCountDown--
		if pollCountDown <= 0 {
			// polling nodes and values takes one call
			nodes, err := svc.PollNodes()
			if err == nil {
				if tdCountDown <= 0 {
					// Every TDInterval publish the full TD's
					err = svc.PublishNodeTDs(nodes)
					tdCountDown = svc.config.TDInterval
				}
				// publish changed values
				err = svc.PublishNodeValues(nodes)
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
	slog.Warn("Stopping OWServer binding")

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
	}
	return svc
}
