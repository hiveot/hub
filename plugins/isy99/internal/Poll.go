// Package internal to poll the ISY for device, node and parameter information
package internal

import (
	"strings"

	"github.com/iotdomain/iotdomain-go/publisher"
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// IsyURL to contact ISY99x gateway
const IsyURL = "http://%s/rest/nodes"

// readIsyNodesValues reads the ISY Node values
// This will run http get on http://address/rest/nodes
// address is the ISY hostname or ip address.
// returns an ISY XML node object and possible error
// func (app *IsyApp) readIsyNodesValues(address string) (*IsyNodes, error) {
// 	isyURL := fmt.Sprintf(IsyURL, address)
// 	isyNodes := IsyNodes{}
// 	err := app.isyAPI.isyRequest(isyURL, &isyNodes)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &isyNodes, nil
// }

// updateDevice updates the node discovery and output value from the provided isy node
func (app *IsyApp) updateDevice(isyNode *IsyNode) {
	nodeHWID := isyNode.Address
	pub := app.pub
	hasInput := false
	outputValue := isyNode.Property.Value
	// take value from simulation as the given node is a static file
	if strings.HasPrefix(app.config.GatewayAddress, "file://") {
		outputValue = app.isyAPI.simulation[isyNode.Address]
	}

	// What node are we dealing with?
	deviceType := types.NodeTypeUnknown
	outputType := types.OutputTypeOnOffSwitch
	switch isyNode.Property.ID {
	case "ST":
		deviceType = types.NodeTypeOnOffSwitch
		outputType = types.OutputTypeOnOffSwitch
		hasInput = true
		if outputValue == "" || outputValue == "DOF" || outputValue == "0" || strings.ToLower(outputValue) == "false" {
			outputValue = "false"
		} else {
			outputValue = "true"
		}
		break
	case "OL":
		deviceType = types.NodeTypeDimmer
		outputType = types.OutputTypeDimmer
		hasInput = true
		break
	case "RR":
		deviceType = types.NodeTypeUnknown
		break
	}
	// Add new discoveries
	node := pub.GetNodeByHWID(nodeHWID)
	if node == nil {
		pub.CreateNode(nodeHWID, types.NodeType(deviceType))
		pub.UpdateNodeConfig(nodeHWID, types.NodeAttrName, &types.ConfigAttr{
			DataType:    types.DataTypeString,
			Description: "Name of ISY node",
			Default:     isyNode.Name,
		})
		pub.UpdateNodeStatus(nodeHWID, map[types.NodeStatus]string{
			types.NodeStatusRunState: types.NodeRunStateReady,
		})
	}

	output := pub.GetOutputByNodeHWID(nodeHWID, outputType, types.DefaultOutputInstance)
	if output == nil {
		// Add an output and optionally an input for the node.
		// Most ISY nodes have only a single sensor. This is a very basic implementation.
		// Is it worth adding multi-sensor support?
		// https://wiki.universal-devices.com/index.php?title=ISY_Developers:API:REST_Interface#Properties
		pub.CreateOutput(nodeHWID, outputType, types.DefaultOutputInstance)
		if hasInput {
			pub.CreateInput(nodeHWID, types.InputType(outputType),
				types.DefaultInputInstance, app.HandleInputCommand)
		}
	}

	//if output.Value() != isyNode.Property.Value {
	//	// this compares 0 with false, so lots of noise
	//	adapter.Logger().Debugf("Isy99Adapter.updateDevice. Update node %s, output %s[%s] from %s to %s",
	//		node.Id, output.IOType, output.Instance, output.Value(), isyNode.Property.Value)
	//}
	// let the adapter decide whether to repeat the same value based on config
	pub.UpdateOutputValue(nodeHWID, outputType, types.DefaultOutputInstance, outputValue)

}

// UpdateDevices discover ISY Nodes from config and ISY gateway
func (app *IsyApp) UpdateDevices() {
	// Discover the ISY nodes
	isyNodes, err := app.isyAPI.ReadIsyNodes()
	if err != nil {
		// Unexpected. What to do now?
		logrus.Warningf("DiscoverNodes: Error reading nodes: %s", err)
		return
	}
	// Update new or changed ISY nodes
	for _, isyNode := range isyNodes.Nodes {
		app.updateDevice(isyNode)
	}
}

// Poll polls the ISY gateway for updates to nodes and sensors
func (app *IsyApp) Poll(pub *publisher.Publisher) {
	_, err := app.ReadGateway()
	if err == nil {
		app.UpdateDevices()
	}
}
