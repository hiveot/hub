package service

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/owserver/service/eds"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
)

// CreateTDFromNode converts the 1-wire node into a TD that describes the node.
// - All attributes will be added as node properties
// - Writable non-sensors attributes are marked as writable configuration
// - Sensors are also added as events.
// - Writable sensors are also added as actions.
func (svc *OWServerBinding) CreateTDFromNode(node *eds.OneWireNode) (tdoc *td.TD) {

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
	// support for setting Thing TD title to the custom name instead of node name
	thingTitle := node.Name
	customTitle := svc.customTitles[thingID]
	if customTitle != "" {
		thingTitle = customTitle
	}
	tdoc = td.NewTD(thingID, thingTitle, deviceType)
	tdoc.UpdateTitleDescription(thingTitle, node.Description)

	// Add a writable 'title' property so consumer can edit the device's title.
	// Since owserver doesn't support naming a device, the title is stored in the state service.
	prop := tdoc.AddProperty(vocab.PropDeviceTitle, "Title", "", vocab.WoTDataTypeString).
		SetAtType(vocab.PropDeviceTitle)
	prop.ReadOnly = false

	// Map node attribute to Thing properties and events
	for attrID, attr := range node.Attr {
		// The AttrInfo table determines what is in the TD as property, event or action
		attrInfo, found := AttrConfig[attrID]

		if !found || attrInfo.Ignore {
			// exclude from the TD
			continue
		}
		if attrInfo.IsProp {
			propType := attrInfo.VocabType
			title := attrInfo.Title
			dataType := attrInfo.DataType

			prop = tdoc.AddProperty(attrID, title, "", dataType).SetAtType(propType)
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
				prop.OneOf = attrInfo.Enum
			}
		}
		if attrInfo.IsEvent {
			var evSchema *td.DataSchema
			// only add data schema if the event carries a value
			if attrInfo.DataType != vocab.WoTDataTypeNone {
				unit, _ := UnitNameVocab[attr.Unit]
				evSchema = &td.DataSchema{
					Type:     attrInfo.DataType,
					Unit:     unit,
					ReadOnly: true,
				}
			}
			// TODO: use a Number/Integerschema for numeric sensors
			tdoc.AddEvent(attrID, attrInfo.Title, "", evSchema).
				SetAtType(attrInfo.VocabType)
		}
		if attrInfo.IsActuator {
			var inputSchema *td.DataSchema
			// only add data schema if the action accepts parameters
			if attrInfo.DataType != vocab.WoTDataTypeNone {
				unit, _ := UnitNameVocab[attr.Unit]
				inputSchema = &td.DataSchema{
					Type:      attrInfo.DataType,
					Unit:      unit,
					ReadOnly:  false,
					WriteOnly: false,
				}
			}
			aff := tdoc.AddAction(attrID, attrInfo.Title, "", inputSchema)
			aff.SetAtType(attrInfo.VocabType)
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

// PublishNodeTD converts a node to TD documents and publishes it to the Hub.
// This returns an error if the publications fail
func (svc *OWServerBinding) PublishNodeTD(node *eds.OneWireNode) (err error) {
	tdi := svc.CreateTDFromNode(node)
	svc.things[tdi.ID] = tdi
	tdJSON, _ := jsoniter.MarshalToString(tdi)
	err = digitwin.ThingDirectoryUpdateTD(&svc.ag.Consumer, tdJSON)
	//err = svc.ag.PubTD(td)
	return err
}

// PublishNodeTDs converts the nodes to TD documents and publishes these to the Hub.
// TD's are stored to be used in publishing its attributes and events.
// This returns an error if one or more publications fail
func (svc *OWServerBinding) PublishNodeTDs(nodes []*eds.OneWireNode) (err error) {
	for _, node := range nodes {
		err2 := svc.PublishNodeTD(node)
		if err2 != nil {
			err = err2
		}
	}
	return err
}
