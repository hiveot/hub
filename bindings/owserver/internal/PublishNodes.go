package internal

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/api/wot"
)

// CreateTDFromNode converts the 1-wire node into a TD that describes the node.
// - All attributes will be added as node properties
// - Writable non-sensors attributes are marked as writable configuration
// - Sensors are also added as events.
// - Writable sensors are also added as actions.
// func (svc *OWServer) CreateTDFromNode(node *eds.OneWireNode) (tdoc *wot.TD) {

// 	// Should we bother with the URI? In HiveOT things have pubsub addresses that include the ID. The ID is not the address.
// 	//thingID := things.CreateThingID(svc.config.ID, node.NodeID, node.DeviceType)
// 	thingID := node.ROMId
// 	if thingID == "" {
// 		thingID = vocab.DeviceNetGateway
// 	}
// 	deviceType := deviceTypeMap[node.Family]
// 	if deviceType == "" {
// 		// unknown device
// 		deviceType = vocab.Device
// 	}
// 	thingTitle := node.Name
// 	tdoc = wot.NewTD(thingID, thingTitle, deviceType)
// 	tdoc.UpdateTitleDescription(thingTitle, node.Description)

// 	// Add a writable 'title' property so consumer can edit the device's title.
// 	// Since owserver doesn't support naming a device, the title is stored in this service.
// 	// The UI should prefer a title property over the TD title.
// 	// how does the dtw know if to forward a title property write request?
// 	//  look into the device's TD
// 	prop := tdoc.AddProperty(wot.WoTTitle, "Title", "", wot.DataTypeString)
// 	prop.SetAtType(vocab.PropDeviceTitle)
// 	prop.ReadOnly = false

// 	// Map node attribute to Thing properties and events
// 	for attrID, attr := range node.Attr {
// 		// The AttrInfo table determines what is in the TD as property, event or action
// 		attrInfo, found := AttrConfig[attrID]

// 		if !found || attrInfo.Ignore {
// 			// exclude from the TD
// 			continue
// 		}
// 		if attrInfo.IsProp {
// 			propType := attrInfo.VocabType
// 			title := attrInfo.Title
// 			dataType := attrInfo.DataType

// 			prop = tdoc.AddProperty(attrID, title, attrInfo.Description, dataType).SetAtType(propType)
// 			unit := attrInfo.Unit
// 			if attr.Unit != "" {
// 				unitID, found := UnitNameVocab[attr.Unit]
// 				if found {
// 					unitInfo := vocab.UnitClassesMap[unitID]
// 					unit = unitInfo.Symbol
// 				}
// 			}
// 			prop.Unit = unit
// 			// non-sensors are attributes. Writable attributes are configuration.
// 			prop.ReadOnly = !attr.Writable
// 			if attrInfo.Enum != nil {
// 				prop.OneOf = attrInfo.Enum
// 			}
// 		}
// 		if attrInfo.IsEvent {
// 			var evSchema *wot.DataSchema
// 			// only add data schema if the event carries a value
// 			if attrInfo.DataType != wot.DataTypeNone {
// 				unit, _ := UnitNameVocab[attr.Unit]
// 				evSchema = &wot.DataSchema{
// 					Type:     attrInfo.DataType,
// 					Unit:     unit,
// 					ReadOnly: true,
// 				}
// 			}
// 			// TODO: use a Number/Integerschema for numeric sensors
// 			tdoc.AddEvent(attrID, attrInfo.Title, attrInfo.Description, evSchema).
// 				SetAtType(attrInfo.VocabType)
// 		}
// 		if attrInfo.IsActuator {
// 			var inputSchema *wot.DataSchema
// 			// only add data schema if the action accepts parameters
// 			if attrInfo.DataType != wot.DataTypeNone {
// 				unit, _ := UnitNameVocab[attr.Unit]
// 				inputSchema = &wot.DataSchema{
// 					Type:      attrInfo.DataType,
// 					Unit:      unit,
// 					ReadOnly:  false,
// 					WriteOnly: false,
// 				}
// 			}
// 			aff := tdoc.AddAction(attrID, attrInfo.Title, attrInfo.Description, inputSchema)
// 			aff.SetAtType(attrInfo.VocabType)
// 		}
// 	}
// 	return tdoc
// }

//func (svc *OWServerBinding) MakeNodePropValues(node *eds.OneWireNode) map[string]string {
//	pv := make(map[string]string)
//	// Map node attribute to Thing properties
//	for attrName, attr := range node.Attr {
//		pv[attrName] = attr.Value
//	}
//
//	return pv
//}

// UpdateThings polls the OWServer gateway for nodes and updates the list of discovered
// 1-wire things on the gateway.
// This writes the TD's of all newly found nodes to the directory. If forceRepublish
// is set then publish the TDs of all known nodes instead.
// This returns a list of newly discovered 1-wire devices Things or an error
// if polling the EDS device failed.
func (svc *OWServerGateway) UpdateThings(onlyChanges bool) ([]*StandardOWThing, error) {
	newThings := make([]*StandardOWThing, 0)
	slog.Info("UpdateThings start")
	nodes, err := svc.edsAPI.PollNodes()
	if err != nil {
		return nil, err
	}
	svc.mux.Lock()
	for _, node := range nodes {
		// TBD: for now this is a 1:1 mapping...
		thingID := node.ROMId
		owt, found := svc.owThings[thingID]
		if !found {
			owt = NewStandardOWThing(thingID, node, svc.edsAPI)
			// relay emitted requests and notifications to this service
			owt.SetRequestSink(svc)
			owt.SetNotificationSink(svc)
			svc.owThings[thingID] = owt
			newThings = append(newThings, owt)
		}
	}
	svc.mux.Unlock()
	if !onlyChanges {
		svc.PublishAllTDs()
	} else {
		// only publish the new TDs
		for _, owt := range newThings {
			tdoc := owt.CreateTD()
			err = svc.WriteTD(tdoc)
			if err != nil {
				return newThings, err
			}
		}
	}
	slog.Info("UpdateThings complete")
	return newThings, nil
}

// PublishNodeTD converts a node to TD documents and publishes it to the Hub.
// This returns an error if the publications fail
// func (svc *OWServerGateway) PublishNodeTD(node *eds.OneWireNode) (err error) {
// 	tdoc := svc.CreateTDFromNode(node)
// 	svc.things[tdoc.ID] = tdoc
// 	svc.WriteTD(tdoc)
// 	//err = svc.ag.UpdateThing(td)
// 	return err
// }

// PublishNodeTDs publishes the TD's of all nodes to the directory.
// This returns an error if one or more publications fail
func (svc *OWServerGateway) PublishAllTDs() (err error) {
	// Need a lock to read the TDs
	svc.mux.RLock()
	tdocs := make([]*wot.TD, 0, len(svc.owThings))
	for _, owt := range svc.owThings {
		tdoc := owt.CreateTD()
		tdocs = append(tdocs, tdoc)
	}
	svc.mux.RUnlock()
	for _, tdoc := range tdocs {
		err = svc.WriteTD(tdoc)
		if err != nil {
			return err
		}
	}
	return nil
}
