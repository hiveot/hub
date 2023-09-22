package internal

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/vocab"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/hiveot/hub/api/go/thing"

	"github.com/hiveot/hub/plugins/owserver/internal/eds"
)

// OWServerBinding is the hub protocol binding plugin for capturing 1-wire OWServer V2 Data
type OWServerBinding struct {
	// Configuration of this protocol binding
	Config OWServerConfig

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
	thingID := binding.Config.BindingID
	td := thing.NewTD(thingID, "OWServer binding", vocab.DeviceTypeBinding)
	// these are configured through the configuration file.
	prop := td.AddProperty(vocab.VocabPollInterval, vocab.VocabPollInterval, "Poll Interval", vocab.WoTDataTypeInteger, "")
	prop.Unit = vocab.UnitNameSecond
	prop.InitialValue = fmt.Sprintf("%d %s", binding.Config.PollInterval, vocab.UnitNameSecond)

	prop = td.AddProperty("tdInterval", vocab.VocabPollInterval, "TD Publication Interval", vocab.WoTDataTypeInteger, "")
	prop.Unit = vocab.UnitNameSecond
	prop.InitialValue = fmt.Sprintf("%d %s", binding.Config.TDInterval, vocab.UnitNameSecond)

	prop = td.AddProperty("valueInterval", vocab.VocabPollInterval, "Value Republication Interval", vocab.WoTDataTypeInteger, "")
	prop.Unit = vocab.UnitNameSecond
	prop.InitialValue = fmt.Sprintf("%d %s", binding.Config.RepublishInterval, vocab.UnitNameSecond)

	prop = td.AddProperty("owServerAddress", vocab.VocabGatewayAddress, "OWServer gateway IP address", vocab.WoTDataTypeString, "")
	prop.InitialValue = fmt.Sprintf("%s", binding.Config.OWServerURL)
	return td
}

// Start the OWServer protocol binding
// This publishes a TD for this binding, starts a background heartbeat.
// This uses the given hub client connection.
func (binding *OWServerBinding) Start() error {

	// Create the adapter for the OWServer 1-wire gateway
	binding.edsAPI = eds.NewEdsAPI(
		binding.Config.OWServerURL, binding.Config.OWServerLogin, binding.Config.OWServerPassword)

	// TODO: restore binding configuration

	td := binding.CreateBindingTD()
	tdDoc, _ := json.Marshal(td)
	err := binding.hc.PubEvent(td.ID, vocab.EventNameTD, tdDoc)
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
	binding.isRunning.Store(false)
	if binding.actionSub != nil {
		binding.actionSub.Unsubscribe()
		binding.actionSub = nil
	}
}

// NewOWServerBinding creates a new OWServer Protocol Binding service
//
//	config holds the configuration of the service
//	devicePubSub holds the publish/subscribe service to use. It will be released on stop.
func NewOWServerBinding(config OWServerConfig, hc hubclient.IHubClient) *OWServerBinding {

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
