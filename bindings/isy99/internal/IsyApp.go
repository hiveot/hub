// Package internal for basic ISY99x Insteon home automation hub access
// This implements common sensors and switches
package internal

import (
	"fmt"
	"time"

	"github.com/iotdomain/iotdomain-go/publisher"
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// ConfigDefaultPollIntervalSec for polling the gateway
const ConfigDefaultPollIntervalSec = 15 * 60

// AppID application name used for configuration file and default publisherID
const appID = "isy99"

// IsyAppConfig with application state, loaded from isy99.yaml
type IsyAppConfig struct {
	GatewayAddress string `yaml:"gatewayAddress"` // gateway IP address
	LoginName      string `yaml:"login"`          // gateway login
	Password       string `yaml:"password"`       // gateway password
	PublisherID    string `yaml:"publisherId"`    // default is app ID
}

// IsyApp adapter main class
// to access multiple gatewways, run additional instances, or modify this code for multiple isyAPI instances
type IsyApp struct {
	config *IsyAppConfig
	pub    *publisher.Publisher
	isyAPI *IsyAPI // ISY gateway access
}

// ReadGateway reads the isy99 gateway device and its nodes
// This returns the ID of the gateway node that was read
func (app *IsyApp) ReadGateway() (gwHWID string, err error) {
	pub := app.pub
	gwHWID = types.NodeIDGateway
	startTime := time.Now()
	isyDevice, err := app.isyAPI.ReadIsyGateway()
	endTime := time.Now()
	latency := endTime.Sub(startTime)

	prevStatus, _ := pub.GetNodeStatus(gwHWID, types.NodeStatusRunState)
	if err != nil {
		// only report this once
		if prevStatus != types.NodeRunStateError {
			// gateway went down
			logrus.Warningf("IsyApp.ReadGateway: ISY99x gateway is no longer reachable on address %s", app.isyAPI.address)
			pub.UpdateNodeStatus(gwHWID, map[types.NodeStatus]string{
				types.NodeStatusRunState:  types.NodeRunStateError,
				types.NodeStatusLastError: "Gateway not reachable on address " + app.isyAPI.address,
			})
		}
		return gwHWID, err
	}

	pub.UpdateNodeStatus(gwHWID, map[types.NodeStatus]string{
		types.NodeStatusRunState:    types.NodeRunStateReady,
		types.NodeStatusLastError:   "Connection restored to address " + app.isyAPI.address,
		types.NodeStatusLatencyMSec: fmt.Sprintf("%d", latency.Milliseconds()),
	})
	logrus.Warningf("Isy99Adapter.ReadGateway: Connection restored to ISY99x gateway on address %s", app.isyAPI.address)

	// Update the info we have on the gateway
	pub.UpdateNodeAttr(gwHWID, map[types.NodeAttr]string{
		types.NodeAttrName:            isyDevice.Configuration.Platform,
		types.NodeAttrSoftwareVersion: isyDevice.Configuration.App + " - " + isyDevice.Configuration.AppVersion,
		types.NodeAttrModel:           isyDevice.Configuration.Product.Description,
		types.NodeAttrManufacturer:    isyDevice.Configuration.DeviceSpecs.Make,
		// types.NodeAttrLocalIP:         isyDevice.network.Interface.IP,
		types.NodeAttrLocalIP: app.isyAPI.address,
		types.NodeAttrMAC:     isyDevice.Configuration.Root.ID,
	})
	return gwHWID, nil
}

// SetupGatewayNode creates the gateway node if it doesn't exist
// This set the default gateway address in its configuration
func (app *IsyApp) SetupGatewayNode(pub *publisher.Publisher) {
	gwID := types.NodeIDGateway
	logrus.Infof("SetupGatewayNode. ID=%s", gwID)

	gatewayNode := pub.GetNodeByHWID(gwID)
	if gatewayNode == nil {
		pub.CreateNode(gwID, types.NodeTypeGateway)
		gatewayNode = pub.GetNodeByHWID(gwID)
	}
	pub.UpdateNodeConfig(gatewayNode.Address, types.NodeAttrLocalIP, &types.ConfigAttr{
		DataType:    types.DataTypeString,
		Description: "ISY gateway IP address",
		Secret:      true,
	})
	pub.UpdateNodeConfig(gatewayNode.Address, types.NodeAttrLoginName, &types.ConfigAttr{
		DataType:    types.DataTypeString,
		Description: "ISY gateway login name",
		Secret:      true,
	})
	pub.UpdateNodeConfig(gatewayNode.Address, types.NodeAttrPassword, &types.ConfigAttr{
		DataType:    types.DataTypeString,
		Description: "ISY gateway login password",
		Secret:      true,
	})

}

// NewIsyApp creates the app
// This creates a node for the gateway
func NewIsyApp(config *IsyAppConfig, pub *publisher.Publisher) *IsyApp {
	app := IsyApp{
		config: config,
		pub:    pub,
		// gatewayNodeAddr: nodes.MakeNodeDiscoveryAddress(pub.Zone, config.PublisherID, GatewayID),
		isyAPI: NewIsyAPI(config.GatewayAddress, config.LoginName, config.Password),
	}
	if app.config.PublisherID == "" {
		app.config.PublisherID = appID
	}
	pub.SetPollInterval(60, app.Poll)
	pub.SetNodeConfigHandler(app.HandleConfigCommand)
	// // Discover the node(s) and outputs. Use default for republishing discovery
	// isyPub.SetDiscoveryInterval(0, app.Discover)
	app.SetupGatewayNode(pub)

	return &app
}

// Run the publisher until the SIGTERM  or SIGINT signal is received
func Run() {
	appConfig := &IsyAppConfig{PublisherID: appID}
	isyPub, _ := publisher.NewAppPublisher(appID, "", appConfig, "", true)

	_ = NewIsyApp(appConfig, isyPub)

	isyPub.Start()
	isyPub.WaitForSignal()
	isyPub.Stop()
}
