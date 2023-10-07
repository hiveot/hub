package service

import (
	"github.com/hiveot/hub/api/go/thing"

	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/plugins/owserver/service/eds"
)

// CreateTDFromNode converts the 1-wire node into a TD that describes the node.
// - All attributes will be added as node properties
// - Writable non-sensors attributes are marked as writable configuration
// - Sensors are also added as events.
// - Writable sensors are also added as actions.
func (binding *OWServerBinding) CreateTDFromNode(node *eds.OneWireNode) (tdoc *thing.TD) {

	// Should we bother with the URI? In HiveOT things have pubsub addresses that include the ID. The ID is not the address.
	//thingID := thing.CreateThingID(binding.Config.ID, node.NodeID, node.DeviceType)
	thingID := node.NodeID

	tdoc = thing.NewTD(thingID, node.Name, node.DeviceType)
	tdoc.UpdateTitleDescription(node.Name, node.Description)

	// Map node attribute to Thing properties
	for attrName, attr := range node.Attr {
		// sensors are added as both properties and events
		if attr.IsSensor {
			evType := attr.VocabType
			var evSchema *thing.DataSchema
			// sensors emit events
			eventID := attrName
			title := attr.Name
			// only add data schema if the event carries a value
			if attr.DataType != vocab.WoTDataTypeNone {
				evSchema = &thing.DataSchema{
					Type:         attr.DataType,
					Unit:         attr.Unit,
					InitialValue: attr.Value,
				}
				if attr.Unit != "" {
					evSchema.InitialValue += " " + attr.Unit
				}
			}
			tdoc.AddEvent(eventID, evType, title, "", evSchema)

		} else if attr.IsActuator {
			// TODO: determine action @type
			var inputSchema *thing.DataSchema
			actionID := attrName
			actionType := attr.VocabType
			// only add data schema if the action accepts parameters
			if attr.DataType != vocab.WoTDataTypeNone {
				inputSchema = &thing.DataSchema{
					Type: attr.DataType,
					Unit: attr.Unit,
				}
			}
			tdoc.AddAction(actionID, actionType, attr.Name, "", inputSchema)
		} else {
			// TODO: determine property @type
			propType := ""
			initialValue := attr.Value
			if attr.Unit != "" {
				initialValue += " " + attr.Unit
			}
			prop := tdoc.AddProperty(attrName, propType, attr.Name, attr.DataType, initialValue)
			prop.Unit = attr.Unit
			// non-sensors are attributes. Writable attributes are configuration.
			if attr.Writable {
				prop.ReadOnly = false
			} else {
				prop.ReadOnly = true
			}
		}
	}
	return
}

// PollNodes polls the OWServer gateway for nodes and property values
func (binding *OWServerBinding) PollNodes() ([]*eds.OneWireNode, error) {
	nodes, err := binding.edsAPI.PollNodes()
	for _, node := range nodes {
		binding.nodes[node.NodeID] = node
	}
	return nodes, err
}

// PublishThings converts the nodes to TD documents and publishes these on the Hub message bus
// This returns an error if one or more publications fail
func (binding *OWServerBinding) PublishThings(nodes []*eds.OneWireNode) (err error) {
	for _, node := range nodes {
		td := binding.CreateTDFromNode(node)
		err2 := binding.hc.PubTD(td)
		if err2 != nil {
			err = err2
		}
	}
	return err
}
