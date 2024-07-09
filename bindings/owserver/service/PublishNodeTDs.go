package service

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/owserver/service/eds"
	thing "github.com/hiveot/hub/lib/things"
)

// CreateTDFromNode converts the 1-wire node into a TD that describes the node.
// - All attributes will be added as node properties
// - Writable non-sensors attributes are marked as writable configuration
// - Sensors are also added as events.
// - Writable sensors are also added as actions.
func CreateTDFromNode(node *eds.OneWireNode) (tdoc *thing.TD) {

	// Should we bother with the URI? In HiveOT things have pubsub addresses that include the ID. The ID is not the address.
	//thingID := things.CreateThingID(svc.config.ID, node.NodeID, node.DeviceType)
	thingID := node.ROMId
	if thingID == "" {
		thingID = vocab.ThingNetGateway
	}
	deviceType := deviceTypeMap[node.Family]
	if deviceType == "" {
		// unknown device
		deviceType = vocab.ThingDevice
	}

	tdoc = thing.NewTD(thingID, node.Name, deviceType)
	tdoc.UpdateTitleDescription(node.Name, node.Description)

	// Map node attribute to Thing properties and events
	for attrID, attr := range node.Attr {
		attrInfo, found := AttrConfig[attrID]

		if !found || attrInfo.Ignore {
			// ignore
			continue
		}
		if attrInfo.IsProp {
			propType := attrInfo.VocabType
			title := attrInfo.Title
			dataType := attrInfo.DataType

			prop := tdoc.AddProperty(attrID, propType, title, dataType)
			unit := attrInfo.Unit
			if attr.Unit != "" {
				unitID, found := UnitNameVocab[attr.Unit]
				if found {
					unitInfo := vocab.UnitClassesMap[unitID]
					unit = unitInfo.Symbol
				}
			}
			prop.Unit = unit
			// non-sensors are attributes. Writable attributes are configuration.
			prop.ReadOnly = !attr.Writable
			if attrInfo.Enum != nil {
				prop.SetEnumValues(attrInfo.Enum)
			}
		}
		if attrInfo.IsEvent {
			var evSchema *thing.DataSchema
			// only add data schema if the event carries a value
			if attrInfo.DataType != vocab.WoTDataTypeNone {
				unit, _ := UnitNameVocab[attr.Unit]
				evSchema = &thing.DataSchema{
					Type:     attrInfo.DataType,
					Unit:     unit,
					ReadOnly: true,
				}
			}
			// Only attributes with a vocab type will be sent as events
			// TODO: use a Number/Integerschema for numeric sensors
			tdoc.AddEvent(attrID, attrInfo.VocabType, attrInfo.Title, "", evSchema)
		}
		if attrInfo.IsActuator {
			var inputSchema *thing.DataSchema
			// only add data schema if the action accepts parameters
			if attrInfo.DataType != vocab.WoTDataTypeNone {
				unit, _ := UnitNameVocab[attr.Unit]
				inputSchema = &thing.DataSchema{
					Type:      attrInfo.DataType,
					Unit:      unit,
					ReadOnly:  false,
					WriteOnly: false,
				}
			}
			tdoc.AddAction(attrID, attrInfo.VocabType, attrInfo.Title, "", inputSchema)
		}
	}
	return
}

//func (svc *OWServerBinding) MakeNodePropValues(node *eds.OneWireNode) map[string]string {
//	pv := make(map[string]string)
//	// Map node attribute to Thing properties
//	for attrName, attr := range node.Attr {
//		pv[attrName] = attr.Value
//	}
//
//	return pv
//}

// PollNodes polls the OWServer gateway for nodes and property values
func (svc *OWServerBinding) PollNodes() ([]*eds.OneWireNode, error) {
	nodes, err := svc.edsAPI.PollNodes()
	for _, node := range nodes {
		svc.nodes[node.ROMId] = node
	}
	return nodes, err
}

// PublishNodeTDs converts the nodes to TD documents and publishes these to the Hub.
// TD's are stored to be used in publishing its attributes and events.
// This returns an error if one or more publications fail
func (svc *OWServerBinding) PublishNodeTDs(nodes []*eds.OneWireNode) (err error) {
	for _, node := range nodes {
		td := CreateTDFromNode(node)
		svc.things[td.ID] = td
		err2 := svc.hc.PubTD(td)
		if err2 != nil {
			err = err2
		} else {
			//props := svc.MakeNodePropValues(node)
			//_ = svc.hc.PubProps(td.ID, props)
		}
	}
	return err
}
