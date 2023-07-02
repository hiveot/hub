package internal

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"golang.org/x/exp/slog"
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

	// Hub CA certificate to validate gateway connection
	//caCert *x509.Certificate

	// Client certificate of this binding
	bindingCert *tls.Certificate

	// client to publish TDs and values and receive actions
	hubClient *hubclient.HubClient

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
	stopChan  chan bool
}

// CreateBindingTD generates a TD document for this binding
func (binding *OWServerBinding) CreateBindingTD() *thing.TD {
	thingID := binding.Config.ID
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
	prop.InitialValue = fmt.Sprintf("%s", binding.Config.OWServerAddress)
	return td
}

// Start the OWServer protocol binding
// This connects to the hub, publishes a TD for this binding, starts a background heartbeat,
// and waits for the context to complete and end the connection.
func (binding *OWServerBinding) Start() error {

	// Create the adapter for the OWServer 1-wire gateway
	binding.edsAPI = eds.NewEdsAPI(
		binding.Config.OWServerAddress, binding.Config.LoginName, binding.Config.Password)

	// TODO: restore binding configuration

	td := binding.CreateBindingTD()
	tdDoc, _ := json.Marshal(td)
	err := binding.hubClient.PubEvent(td.ID, vocab.EventNameTD, tdDoc)
	if err != nil {
		return err
	}

	err = binding.hubClient.SubActions(binding.HandleActionRequest)
	if err != nil {
		return err
	}
	binding.isRunning.Store(true)

	go binding.heartBeat()

	slog.Info("Service OWServer startup completed")

	<-binding.stopChan
	binding.isRunning.Store(false)
	return nil
}

// Stop the heartbeat and remove subscriptions
func (binding *OWServerBinding) Stop() {
	binding.stopChan <- true
}

// NewOWServerBinding creates a new OWServer Protocol Binding service
//
//	config holds the configuration of the service
//	devicePubSub holds the publish/subscribe service to use. It will be released on stop.
func NewOWServerBinding(config OWServerConfig, hubClient *hubclient.HubClient) *OWServerBinding {

	// these are from hub configuration
	pb := &OWServerBinding{
		hubClient: hubClient,
		values:    make(map[string]map[string]NodeValueStamp),
		nodes:     make(map[string]*eds.OneWireNode),
		isRunning: atomic.Bool{},
		stopChan:  make(chan bool),
	}
	pb.Config = config

	return pb
}
