package internal

import (
	"fmt"
	"log/slog"
	"math"
	"strconv"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hub/bindings/owserver/internal/eds"
)

// StandardOWThing represents a 1-wire device available through the OWServer gateway
type StandardOWThing struct {
	*thing.ExposedThing

	// the API for access the node via the gateway
	edsAPI *eds.EdsAPI

	node *eds.OneWireNode
	td   *wot.TD
}

// Convert the attribute value to published value
func (owt *StandardOWThing) ConvertValue(
	attrValue string, info AttrConversion) any {

	var pubValue any = attrValue

	// parse all data values to their native types and round as per conversion spec
	switch info.DataType {
	case wot.DataTypeNumber:
		valueFloat, err := strconv.ParseFloat(attrValue, 32)
		if err != nil {
			return attrValue
		}
		prec := math.Pow10(info.Precision)
		roundedInt := int(valueFloat * prec)
		// FIXME: check for changenotify
		pubValue = float64(roundedInt) / prec

	case wot.DataTypeInteger, wot.DataTypeUnsignedInt:
		valueInt64, err := strconv.ParseInt(attrValue, 10, 32)
		if err != nil {
			return attrValue
		}
		// FIXME: check for changenotify
		pubValue = int(valueInt64)

	case wot.DataTypeBool:
		valueBool, err := strconv.ParseBool(attrValue)
		if err != nil {
			return attrValue
		}
		pubValue = valueBool
	}
	return pubValue
}

// Return the TD for this 1-wire device
func (owt *StandardOWThing) CreateTD() *wot.TD {
	node := owt.node

	// Should we bother with the URI? In HiveOT things have pubsub addresses that include the ID. The ID is not the address.
	//thingID := things.CreateThingID(svc.config.ID, node.NodeID, node.DeviceType)
	thingID := node.ROMId
	deviceType := deviceTypeMap[node.Family]
	if deviceType == "" {
		// general purpose device
		deviceType = vocab.Device
	}
	thingTitle := node.Name
	tdoc := wot.NewTD(thingID, thingTitle, deviceType)
	tdoc.UpdateTitleDescription(thingTitle, node.Description)

	// Add a writable 'title' property so consumer can edit the device's title.
	// Since owserver doesn't support naming a device, the title is stored in
	// the service.
	prop := tdoc.AddProperty(wot.WoTTitle, "Title", "", wot.DataTypeString)
	prop.SetAtType(vocab.PropDeviceTitle)
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

			prop = tdoc.AddProperty(attrID, title, attrInfo.Description, dataType).SetAtType(propType)
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
			var evSchema *wot.DataSchema
			// only add data schema if the event carries a value
			if attrInfo.DataType != wot.DataTypeNone {
				unit, _ := UnitNameVocab[attr.Unit]
				evSchema = &wot.DataSchema{
					Type:     attrInfo.DataType,
					Unit:     unit,
					ReadOnly: true,
				}
			}
			// TODO: use a Number/Integerschema for numeric sensors
			tdoc.AddEvent(attrID, attrInfo.Title, attrInfo.Description, evSchema).
				SetAtType(attrInfo.VocabType)
		}
		if attrInfo.IsActuator {
			var inputSchema *wot.DataSchema
			// only add data schema if the action accepts parameters
			if attrInfo.DataType != wot.DataTypeNone {
				unit, _ := UnitNameVocab[attr.Unit]
				inputSchema = &wot.DataSchema{
					Type:      attrInfo.DataType,
					Unit:      unit,
					ReadOnly:  false,
					WriteOnly: false,
				}
			}
			aff := tdoc.AddAction(attrID, attrInfo.Title, attrInfo.Description, inputSchema)
			aff.SetAtType(attrInfo.VocabType)
		}
	}
	return tdoc
}

// HandleRequest handles action and property requests for Things managed
// by the OWServer gateway.
//
// For 1-wire, configuration and actions are the same thing
func (owt *StandardOWThing) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	var err error

	slog.Info("HandleRequest",
		slog.String("op", req.Operation),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("payload", req.ToString(20)),
	)

	// Custom config. Change device title.
	if req.Operation == wot.OpWriteProperty {
		// This is a node update
		valueStr := req.ToString(0)

		err := owt.WriteNode(req.Name, valueStr)
		if err != nil {
			return err
		}

		// poll the result a few times until the requested status is reached or until timeout.
		// newValue, hasUpdated := owt.WaitForValueUpdate(req, MaxUpdateWaitTime)
		// if !hasUpdated {
		// err = fmt.Errorf("Node didn't update property within %d seconds", MaxUpdateWaitTime)
		// }
		// complete or fail the request
		resp := req.CreateResponse(nil, nil)
		replyTo(resp)

		// always notify subscribers of the update
		go owt.PubProperty(req.Name, req.Input, false)

	} else if req.Operation == wot.OpInvokeAction {
		// no actions supported
		err = fmt.Errorf("Actions not supported")
	} else {
		// let the base handle read requests
		err = owt.ExposedThing.HandleRequest(req, replyTo)
	}
	return err

}

// Update the node properties
//
//	node with the property values
//	deviceTitle with the custom title or "" if no custom title
//	onlyChanges publish only the changed properties
func (owt *StandardOWThing) UpdateProperties(
	node *eds.OneWireNode, deviceTitle string, onlyChanges bool) {

	if deviceTitle != "" {
		owt.PubProperty(vocab.PropDeviceTitle, deviceTitle, onlyChanges)
	}
	slog.Info("UpdateProperties",
		"ROMId", node.ROMId, "Name", node.Name, "onlyChanges", onlyChanges)

	// first and unknown values are always changed
	for attrID, attr := range node.Attr {

		// collect changes to property values
		info, found := AttrConfig[attrID]
		if !found {
			// these values are unlisted and ignored
			//slog.Info("Ignored value", "name", attrID, "value", attr.Value)
		} else if info.Ignore {
			// do nothing when ignore is set
		} else {
			// adjust the precision
			pubValue := owt.ConvertValue(attr.Value, info)
			if info.IsEvent {
				owt.PubEvent(attrID, pubValue, onlyChanges)
			} else if info.IsProp {
				owt.PubProperty(attrID, pubValue, onlyChanges)
			}
		}
	}
}

// WriteNode validates and writes the new value to the node
// in 1-wire actions and configuration is the same thing
func (owt *StandardOWThing) WriteNode(propName string, valueStr string) (err error) {

	// This is a node update
	attr, found := owt.node.Attr[propName]
	if !found {
		// delivery completed with error
		err = fmt.Errorf("node '%s' found but it doesn't have an attribute '%s'",
			owt.GetID(), propName)
		return err
	} else if !attr.Writable {
		// delivery completed with error
		err = fmt.Errorf("node '%s' attribute '%s' is read-only",
			owt.GetID(), propName)
		return err
	}

	//if valueStr is a boolean then convert it to a number as 1-wire doesn't have booleans
	if valueStr == "true" {
		valueStr = "1"
	} else if valueStr == "false" {
		valueStr = "0"
	}

	err = owt.edsAPI.WriteNode(owt.GetID(), propName, valueStr)
	return err
}

// Create a new Thing instance for a 1-wire device
// This is called 'standard' as it covers most 1-wire devices.
func NewStandardOWThing(thingID string, node *eds.OneWireNode, edsAPI *eds.EdsAPI) *StandardOWThing {
	owt := &StandardOWThing{
		ExposedThing: thing.NewExposedThing(thingID, nil),
		edsAPI:       edsAPI,
		node:         node,
	}
	return owt
}
