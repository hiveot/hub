package service

import (
	"fmt"
	"github.com/hiveot/hub/bindings/owserver/config"
	"github.com/hiveot/hub/bindings/owserver/service/eds"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
	"sync"
	"time"
)

// OWServerBinding is the hub protocol binding plugin for capturing 1-wire OWServer V2 Data
type OWServerBinding struct {
	// Configuration of this protocol binding
	config *config.OWServerConfig

	// EDS OWServer client API
	edsAPI *eds.EdsAPI

	// hub client to publish TDs and values and receive actions
	hc *hubclient.HubClient

	// track the last value for change detection
	// map of [node/device ID] [attribute name] value
	values map[string]map[string]NodeValueStamp

	// nodes by deviceID/thingID
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
	td := things.NewTD(thingID, "OWServer svc", vocab.DeviceTypeBinding)
	// these are configured through the configuration file.
	prop := td.AddProperty(vocab.VocabPollInterval, vocab.VocabPollInterval, "Poll Interval", vocab.WoTDataTypeInteger, "")
	prop.Unit = vocab.UnitNameSecond
	prop.InitialValue = fmt.Sprintf("%d %s", svc.config.PollInterval, vocab.UnitNameSecond)

	prop = td.AddProperty("tdInterval", vocab.VocabPollInterval, "TD Publication Interval", vocab.WoTDataTypeInteger, "")
	prop.Unit = vocab.UnitNameSecond
	prop.InitialValue = fmt.Sprintf("%d %s", svc.config.TDInterval, vocab.UnitNameSecond)

	prop = td.AddProperty("valueInterval", vocab.VocabPollInterval, "Value Republication Interval", vocab.WoTDataTypeInteger, "")
	prop.Unit = vocab.UnitNameSecond
	prop.InitialValue = fmt.Sprintf("%d %s", svc.config.RepublishInterval, vocab.UnitNameSecond)

	prop = td.AddProperty("owServerAddress", vocab.VocabGatewayAddress, "OWServer gateway IP address", vocab.WoTDataTypeString, "")
	prop.InitialValue = fmt.Sprintf("%s", svc.config.OWServerURL)
	return td
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
					// Every TDInterval update the TD's and submit all properties
					// create ExposedThing's as they are discovered
					err = svc.PublishThings(nodes)
					tdCountDown = svc.config.TDInterval
				}

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
	slog.Warn("Stopping OWServer svc")

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
