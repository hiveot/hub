package service

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/thing"
	vocab2 "github.com/hiveot/hub/lib/vocab"
	"github.com/hiveot/hub/plugins/owserver/config"
	"github.com/hiveot/hub/plugins/owserver/service/eds"
	"log/slog"
	"sync"
	"sync/atomic"
)

// OWServerBinding is the hub protocol binding plugin for capturing 1-wire OWServer V2 Data
type OWServerBinding struct {
	// Configuration of this protocol binding
	Config config.OWServerConfig

	// EDS OWServer client API
	edsAPI *eds.EdsAPI

	// hub client to publish TDs and values and receive actions
	hc hubclient.IHubClient

	// subscription to actions
	actionSub hubclient.ISubscription

	// track the last value for change detection
	// map of [node/device ID] [attribute name] value
	values map[string]map[string]NodeValueStamp

	// nodes by deviceID/thingID
	nodes map[string]*eds.OneWireNode

	// Map of previous node values [nodeID][attrName]value
	// nodeValues map[string]map[string]string

	// flag, this service is up and isRunning
	isRunning atomic.Bool
	mu        sync.Mutex
}

// CreateBindingTD generates a TD document for this binding
func (binding *OWServerBinding) CreateBindingTD() *thing.TD {
	thingID := binding.hc.ClientID()
	td := thing.NewTD(thingID, "OWServer binding", vocab2.DeviceTypeBinding)
	// these are configured through the configuration file.
	prop := td.AddProperty(vocab2.VocabPollInterval, vocab2.VocabPollInterval, "Poll Interval", vocab2.WoTDataTypeInteger, "")
	prop.Unit = vocab2.UnitNameSecond
	prop.InitialValue = fmt.Sprintf("%d %s", binding.Config.PollInterval, vocab2.UnitNameSecond)

	prop = td.AddProperty("tdInterval", vocab2.VocabPollInterval, "TD Publication Interval", vocab2.WoTDataTypeInteger, "")
	prop.Unit = vocab2.UnitNameSecond
	prop.InitialValue = fmt.Sprintf("%d %s", binding.Config.TDInterval, vocab2.UnitNameSecond)

	prop = td.AddProperty("valueInterval", vocab2.VocabPollInterval, "Value Republication Interval", vocab2.WoTDataTypeInteger, "")
	prop.Unit = vocab2.UnitNameSecond
	prop.InitialValue = fmt.Sprintf("%d %s", binding.Config.RepublishInterval, vocab2.UnitNameSecond)

	prop = td.AddProperty("owServerAddress", vocab2.VocabGatewayAddress, "OWServer gateway IP address", vocab2.WoTDataTypeString, "")
	prop.InitialValue = fmt.Sprintf("%s", binding.Config.OWServerURL)
	return td
}

// Start the OWServer protocol binding
// This publishes a TD for this binding, starts a background heartbeat.
// This uses the given hub client connection.
func (binding *OWServerBinding) Start() error {
	slog.Warn("Starting OWServer binding")
	// Create the adapter for the OWServer 1-wire gateway
	binding.edsAPI = eds.NewEdsAPI(
		binding.Config.OWServerURL, binding.Config.OWServerLogin, binding.Config.OWServerPassword)

	// TODO: restore binding configuration

	td := binding.CreateBindingTD()
	tdDoc, _ := json.Marshal(td)
	err := binding.hc.PubEvent(td.ID, vocab2.EventNameTD, tdDoc)
	if err != nil {
		return err
	}

	binding.actionSub, err = binding.hc.SubActions(
		"", binding.HandleActionRequest)
	if err != nil {
		return err
	}
	binding.isRunning.Store(true)

	go binding.heartBeat()

	slog.Info("Service OWServer startup completed")

	return nil
}

// Stop the heartbeat and remove subscriptions
// This does NOTE close the given hubclient connection.
func (binding *OWServerBinding) Stop() {
	slog.Warn("Stopping OWServer binding")
	binding.isRunning.Store(false)
	if binding.actionSub != nil {
		binding.actionSub.Unsubscribe()
		binding.actionSub = nil
	}
}

// NewOWServerBinding creates a new OWServer Protocol Binding service
//
//	config holds the configuration of the service
//	hc is the connection with the hubClient to use.
func NewOWServerBinding(config config.OWServerConfig, hc hubclient.IHubClient) *OWServerBinding {

	// these are from hub configuration
	pb := &OWServerBinding{
		hc:        hc,
		values:    make(map[string]map[string]NodeValueStamp),
		nodes:     make(map[string]*eds.OneWireNode),
		isRunning: atomic.Bool{},
		//stopChan:  make(chan bool),
	}
	pb.Config = config

	return pb
}
