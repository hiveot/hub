package service

import (
	"github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/bindings/owserver/service/eds"
	thing2 "github.com/hiveot/hub/lib/things"
)

// CreateTDFromNode converts the 1-wire node into a TD that describes the node.
// - All attributes will be added as node properties
// - Writable non-sensors attributes are marked as writable configuration
// - Sensors are also added as events.
// - Writable sensors are also added as actions.
func (svc *OWServerBinding) CreateTDFromNode(node *eds.OneWireNode) (tdoc *thing2.TD) {

	// Should we bother with the URI? In HiveOT things have pubsub addresses that include the ID. The ID is not the address.
	//thingID := things.CreateThingID(svc.config.ID, node.NodeID, node.DeviceType)
	thingID := node.NodeID

	tdoc = thing2.NewTD(thingID, node.Name, node.DeviceType)
	tdoc.UpdateTitleDescription(node.Name, node.Description)

	// Map node attribute to Thing properties
	for attrName, attr := range node.Attr {
		// sensors are added as both properties and events
		if attr.IsSensor {
			evType := attr.VocabType
			var evSchema *thing2.DataSchema
			// sensors emit events
			eventID := attrName
			title := attr.Name
			// only add data schema if the event carries a value
			if attr.DataType != vocab.WoTDataTypeNone {
				evSchema = &thing2.DataSchema{
					Type: attr.DataType,
					Unit: attr.Unit,
					//InitialValue: attr.Value,
				}
				if attr.Unit != "" {
					//evSchema.InitialValue += " " + attr.Unit
				}
			}
			tdoc.AddEvent(eventID, evType, title, "", evSchema)

		} else if attr.IsActuator {
			// TODO: determine action @type
			var inputSchema *thing2.DataSchema
			actionID := attrName
			actionType := attr.VocabType
			// only add data schema if the action accepts parameters
			if attr.DataType != vocab.WoTDataTypeNone {
				inputSchema = &thing2.DataSchema{
					Type: attr.DataType,
					Unit: attr.Unit,
				}
			}
			tdoc.AddAction(actionID, actionType, attr.Name, "", inputSchema)
		} else {
			// TODO: determine property @type
			propType := ""
			prop := tdoc.AddProperty(attrName, propType, attr.Name, attr.DataType)
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

func (svc *OWServerBinding) MakeNodeProps(node *eds.OneWireNode) map[string]string {
	pv := make(map[string]string)
	// Map node attribute to Thing properties
	for attrName, attr := range node.Attr {
		pv[attrName] = attr.Value
	}

	return pv
}

// PollNodes polls the OWServer gateway for nodes and property values
func (svc *OWServerBinding) PollNodes() ([]*eds.OneWireNode, error) {
	nodes, err := svc.edsAPI.PollNodes()
	for _, node := range nodes {
		svc.nodes[node.NodeID] = node
	}
	return nodes, err
}

// PublishThings converts the nodes to TD documents and publishes these on the Hub message bus
// This returns an error if one or more publications fail
func (svc *OWServerBinding) PublishThings(nodes []*eds.OneWireNode) (err error) {
	for _, node := range nodes {
		td := svc.CreateTDFromNode(node)
		err2 := svc.hc.PubTD(td)
		if err2 != nil {
			err = err2
		} else {
			props := svc.MakeNodeProps(node)
			_ = svc.hc.PubProps(td.ID, props)
		}
	}
	return err
}
