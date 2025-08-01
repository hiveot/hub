import {
    NodeStatus,
    ZWaveNode,
    ZWavePlusNodeType,
    ZWavePlusRoleType
} from "zwave-js";
import type {
    TranslatedValueID,
    ValueMetadataBoolean,
    ValueMetadataNumeric,
    ValueMetadataString,
} from "zwave-js";
import {CommandClasses, InterviewStage} from '@zwave-js/core';

import TD, {type ActionAffordance, type EventAffordance, type PropertyAffordance} from "../hivelib/wot/TD.ts";

import * as vocab from "../hivelib/api/vocab/vocab.js";
import  DataSchema from "../hivelib/wot/dataSchema.ts";
import {
    UnitMilliSecond,
    WoTDataTypeArray,
    WoTDataTypeBool,
    WoTDataTypeNone,
    WoTDataTypeNumber,
    WoTDataTypeString, WoTTitle
} from "../hivelib/api/vocab/vocab.js";

import type ZWAPI from "./ZWAPI.ts";
import logVid from "./logVid.ts";
import getAffordanceFromVid, { type VidAffordance} from "./getAffordanceFromVid.ts";
import {getDeviceType} from "./getDeviceType.ts";


// Add the ZWave value data to the TD as an action
// Actions in this binding have the same output schema as input schema
function addAction(td: TD, node: ZWaveNode, vid: TranslatedValueID, va: VidAffordance): ActionAffordance {
    // let vidMeta = node.getValueMetadata(vid)

    // actions without input have no schema. How to identify these?
    const schema = new DataSchema()
    SetDataSchema(schema, node, vid)
    // va.title takes precendence if provided
    const action = td.AddAction(
        va.name, va.title || schema.title ||  va.name, schema.description, schema)
        .setVocabType(va.atType)

    if (action.input) {
        // The VID title, description belongs to the action, not the schema
        action.input.title = undefined
        action.input.description = undefined
        action.input.readOnly = false
        // all actions have the same output schema as the input schema
        action.output = action.input
        action.output.readOnly = false
    }
    return action
}

// Add the ZWave value data to the TD as an attribute property
function addProperty(tdi: TD, node: ZWaveNode, vid: TranslatedValueID,  va: VidAffordance): PropertyAffordance {

    const prop = tdi.AddProperty(
        va.name, va.name, "", WoTDataTypeNone, va.atType)

    // if va is an action then this property is readonly
    // SetDataSchema also sets the title, data type and read-only
    SetDataSchema(prop, node, vid)
    if (va.title) {
        prop.title = va.title
    }
    // action status is also sent as read-only property
    if (va?.affType === "action") {
        prop.readOnly = true
    }
    return prop
}

// Add the ZWave VID to the TD as a configuration property
function addConfig(tdi: TD, node: ZWaveNode, vid: TranslatedValueID, va: VidAffordance): PropertyAffordance {
    const prop = tdi.AddProperty(
        va.name, va.title || va.name,  "",WoTDataTypeNone, va.atType)
    prop.readOnly = false
    // SetDataSchema also sets the title and data type
    SetDataSchema(prop, node, vid)
    return prop
}

// Add the ZWave VID to the TD as an event
function addEvent(td: TD, node: ZWaveNode, vid: TranslatedValueID,  va: VidAffordance): EventAffordance {

    const schema = new DataSchema()
    SetDataSchema(schema, node, vid)

    const ev = td.AddEvent(
        va.name, va.title || schema.title || va.name, schema.description, schema)
        .setVocabType(va.atType)

    // SetDataSchema use the dataschema title, not the event data title
    if (ev.data) {
        ev.data.title = undefined
        ev.data.description = undefined
    }
    return ev
}

/**
createNodeTD converts a ZWave Node into a WoT TD document
- extract available node attributes and configuration
- convert ZWave vocabulary to WoT/HiveOT vocabulary
- build a TD document containing properties, events and actions
- if this is the controller node, add controller attributes and actions
@param zwapi wrapper around the zwave driver
@param node the zwave node definition
@param vidLogFD optional file handle to log VID info to CSV for further analysis
@param maxNrScenes limit the nr of scenes in the TD as it would bloat the TD to a massive size.
 */
export default function createNodeTD(zwapi: ZWAPI, node: ZWaveNode, vidLogFD: number | undefined, maxNrScenes: number): TD {

    //--- Step 1: TD definition
    const deviceID = zwapi.getDeviceID(node.id)
    const deviceType = getDeviceType(node)

    // node name is the customizable name (if provided) used as device title and
    // persisted in the driver using CC 119-name.
    // device name: https://zwave-js.github.io/zwave-js/#/api/node?id=name
    let title = node.name
    if (!title) {
        title = node.label || deviceID
        if (node.deviceConfig?.description) {
            title += " " + node.deviceConfig?.description
        }
    }
    // node.label is the manufacturer provided device label
    let description = (node.label || deviceID) + ", " + deviceType
    if (node.deviceConfig) {
        description = node.deviceConfig.manufacturer + " " + description + ", " + node.deviceConfig.description
    }

    // if (node.deviceConfig) {
    //     description = node.deviceConfig.description
    // }
    const tdi = new TD(deviceID, deviceType, title, description);

    //--- Step 2: Add read-only attributes that are common to many nodes
    // since none of these have standard property names, use the ZWave name instead.
    // these names must match those used in parseNodeValues()
    tdi.AddProperty("associationCount", "Association Count","",WoTDataTypeNumber, );

    tdi.AddPropertyIf(node.canSleep,
        "canSleep","Can Sleep", "Device sleeps to conserve battery", WoTDataTypeBool);
    if (node.deviceClass) {
        // tdi.AddPropertyIf( node.deviceClass.basic,"deviceClassBasic",
        //     node.deviceClass.basic.label, "", WoTDataTypeString);
        tdi.AddPropertyIf(node.deviceClass.generic,
            "deviceClassGeneric", node.deviceClass.generic.label,"", WoTDataTypeString);
        tdi.AddPropertyIf(node.deviceClass.specific.label,
            "deviceClassSpecific", node.deviceClass.specific.label,"", WoTDataTypeString);
        // this.setIf("supportedCCs", node.deviceClass.generic.supportedCCs);
    }

    tdi.AddPropertyIf(node.deviceDatabaseUrl,
        "deviceDatabaseURL","Database URL", "Link to database with device information", WoTDataTypeString,);
    tdi.AddProperty(vocab.PropDeviceDescription, "Description", "", WoTDataTypeString, vocab.PropDeviceDescription);

    tdi.AddProperty("endpointCount", "Endpoints", "Number of endpoints in this node",WoTDataTypeNumber);
    tdi.AddProperty(vocab.PropDeviceFirmwareVersion, "Device firmware version","", WoTDataTypeString, vocab.PropDeviceFirmwareVersion);

    tdi.AddPropertyIf(node.getHighestSecurityClass(),
        "highestSecurityClass", "Security Class", "",WoTDataTypeString);
    tdi.AddPropertyIf(node.interviewAttempts,
        "interviewAttempts", "Nr interview attempts","",WoTDataTypeNumber);
    if (node.interviewStage) {
        tdi.AddProperty("interviewStage", "Device Interview Stage", "",
            WoTDataTypeString).SetAsEnum(InterviewStage)
    }
    tdi.AddProperty("isFailedNode", "Is Dead Node",
        "Node was marked as failed in the controller", WoTDataTypeBool);
    tdi.AddProperty("isListening", "Is Listening",
        "The device is always listening and does not sleep", WoTDataTypeBool);

    tdi.AddPropertyIf(node.isSecure, "isSecure","Secured",
        "Device communicates securely with controller",WoTDataTypeBool );
    tdi.AddPropertyIf(node.isRouting, "isRouting","Routing Device",
        "Device support message routing/forwarding (if listening)", WoTDataTypeBool );

    tdi.AddPropertyIf(node.isControllerNode, "isControllerNode",
        "Controller Node", "Device is a ZWave controller", WoTDataTypeBool);
    tdi.AddPropertyIf(node.keepAwake, "keepAwake","Keep Awake",
        "Device stays awake a bit longer before sending it to sleep",WoTDataTypeBool)
    tdi.AddPropertyIf(node.label, "nodeLabel","Node Label",
        "Manufacturer device label",  WoTDataTypeString,vocab.PropDeviceModel);
    tdi.AddProperty("lastSeen","Last Seen",
        "Time this node last sent or received a command", WoTDataTypeString)


    // tdi.AddPropertyIf(node.manufacturerId,
    //     "manufacturerId","Manufacturer ID", "", WoTDataTypeString);

    tdi.AddPropertyIf(node.deviceConfig?.manufacturer,
        "manufacturerName", "Manufacturer Name","", WoTDataTypeString,vocab.PropDeviceMake);

    tdi.AddPropertyIf(node.maxDataRate,
        "maxDataRate","Max data rate", "Device maximum communication data rate", WoTDataTypeNumber);
    if (node.nodeType) {
        // td.AddProperty("nodeType", "ZWave node type",
        //     "",WoTDataTypeNumber)
        // td.AddProperty("nodeTypeName", "ZWave node type name",
        //     "",WoTDataTypeString).SetAsEnum(NodeType)
    }
    // tdi.AddPropertyIf(node.productId,
    //     "productId", "Product ID", "", WoTDataTypeNumber);
    // tdi.AddPropertyIf(node.productType,
    //     "productType","Product Type", "", WoTDataTypeNumber);
    tdi.AddPropertyIf(node.protocolVersion,
        "protocolVersion","ZWave protocol version", "", WoTDataTypeString);

    if (node.statistics) {
        tdi.AddProperty("commandsTX", "TX commands", "Nr of successful commands sent sent to this node", WoTDataTypeNumber)
        tdi.AddProperty("commandsRX", "RX commands", "Nr of commands received from this node, including responses to sent commands", WoTDataTypeNumber)
        tdi.AddProperty("commandsDroppedRX", "RX commands dropped", "Nr of commands sent by the node that were dropped by the controller", WoTDataTypeNumber)
        tdi.AddProperty("commandsDroppedTX", "TX commands dropped", "Nr of commands that failed to sent to this node", WoTDataTypeNumber)

        tdi.AddProperty("rssi", "RSSI", "Average received signal strength indicator in dBm. A higher value indicates jamming", WoTDataTypeNumber)
            .unit = "dBm"

        tdi.AddProperty("rtt", "RTT (latency)", "Moving average of the round-trip-time of commands sent to this node", WoTDataTypeNumber)
            .unit = UnitMilliSecond
        tdi.AddProperty("timeoutResponse", "Nr Get timeouts", "Number of Get-type commands where the response did not come in time", WoTDataTypeNumber)
    }
    tdi.AddPropertyIf(node.supportedDataRates,
        "supportedDataRates","ZWave Data Speed", "", WoTDataTypeString);

    tdi.AddPropertyIf(node.sdkVersion,
        vocab.PropDeviceSoftwareVersion,"SDK version", "", WoTDataTypeString, vocab.PropDeviceSoftwareVersion);
    if (node.status) {
        tdi.AddProperty(vocab.PropDeviceStatus, "Node status", "", WoTDataTypeNumber, vocab.PropDeviceStatus)
            .SetAsEnum(NodeStatus)
    }
    tdi.AddPropertyIf(node.supportedDataRates,
        "supportedDataRates","ZWave Data Speed", "", WoTDataTypeString);

    tdi.AddPropertyIf(node.userIcon,
        "userIcon","Icon","", WoTDataTypeString);

    // show whether this is ZWave+
    if (node.zwavePlusNodeType) {
        tdi.AddProperty("zwavePlusNodeType", "ZWave+ Node Type", "", WoTDataTypeNumber)
        const prop = tdi.AddProperty(
            "zwavePlusNodeTypeName", "ZWave+ Node Type Name", "", WoTDataTypeString)
        if (node.zwavePlusNodeType != undefined) {
            prop.SetAsEnum(ZWavePlusNodeType)
        } else {
            prop.description = "Z-Wave+ Command Class is not supported"
        }
    }
    if (node.zwavePlusRoleType) {
        tdi.AddProperty("zwavePlusRoleType", "ZWave+ Role Type", "",  WoTDataTypeNumber)
        tdi.AddProperty("zwavePlusRoleTypeName", "ZWave+ Role Type Name", "", WoTDataTypeString)
            .SetAsEnum(ZWavePlusRoleType)
    }
    tdi.AddPropertyIf(node.zwavePlusVersion,
        "zwavePlusVersion","Z-Wave+ Version", "",WoTDataTypeNumber);

    // Node actions

    let act = tdi.AddAction("checkLifelineHealth", "Check connection health",
        "Initiates tests to check the health of the connection between the controller and this node. " +
        "This should NOT be done while there is a lot of traffic on the network because it will negatively impact the test results")
    act.output = new DataSchema({
        "title": "Health Rating",
        "description": "Health rating 0-10",
        "type": WoTDataTypeNumber,
    })

    act = tdi.AddAction("checkRouteHealth", "Check Route Health",
        "Check connection health between this node and another node."+
        "This should NOT be done while there is a lot of traffic on the network because it will negatively impact the test results",
        new DataSchema({title: "Node Nr", type: WoTDataTypeNumber}))
    act.output = new DataSchema({
        "title": "Rating",
        "description": "Health rating 0-10",
        "type": WoTDataTypeNumber,
    })

    // https://zwave-js.github.io/zwave-js/#/api/controller?id=getnodeneighbors
    act = tdi.AddAction("getNodeNeighbors",  "Update Neighbors",
        "Request update of this node's neighbors list",
    )
    act.comment = "https://zwave-js.github.io/zwave-js/#/api/controller?id=getnodeneighbors"
    act.output = new DataSchema({
        "title": "Neighbors",
        "description": "Obtain the known list of neighbors of this node.",
        "type": WoTDataTypeArray,
    })
    // ping result is also updated as a property
    tdi.AddProperty("getNodeNeighbors", "Neighbors", "The last result of reading the list of connected neighbors.", WoTDataTypeNumber)

    act = tdi.AddAction("ping", "Ping", "Ping the device")
    act.output = new DataSchema({
        "title": "Duration",
        "description": "Delay in msec before a response is received.",
        "type": WoTDataTypeNumber,
        "unit": vocab.UnitMilliSecond
    })

    // controllers use beginRebuildingRoutes instead
    if (!node.isControllerNode) {
        act = tdi.AddAction("rebuildNodeRoutes", "Rebuild node routes",
            "Rebuilds routes for a single alive node in the network, updating the neighbor list and " +
            "assigning fresh routes to association targets. "
        )
        act.comment = "https://zwave-js.github.io/zwave-js/#/api/controller?id=rebuildnoderoutes"
    }
    act = tdi.AddAction("refreshInfo",  "Refresh Device Info",
        "Resets (almost) all information about this node and forces a fresh interview. " +
        "Ignored when interview is in progress. After this action, the node will "+
        "no longer be ready. This can take a long time.")
    act.comment = "https://zwave-js.github.io/zwave-js/#/api/node?id=refreshinfo"


    tdi.AddAction("refreshValues", "Refresh Device Values",
        "Refresh all non-static sensor and actuator values. " +
        "Use sparingly. This can take a long time and generate a lot of traffic.")
    // todo: what type of response is expected: progress status updates until completed

    if (!node.isControllerNode) {
        tdi.AddAction("removeFailedNode", "Remove failed node",
            "Remove this node from the network if status is failed"
        )
    }
    //--- Step 4: add properties, events, and actions from the ValueIDs

    //--- Use the CC 119- (config) for device title and location
    // Currently the name and location VID (CC 119) is only included when a
    // name and location is set, so create the Vid here manually to ensure it is
    // always there.
    // Reading this CC 119 does work and returns the value set with node.name and
    // node.location.
    const titleCC = CommandClasses["Node Naming and Location"] // 119
    const nameVid = {
        commandClass: titleCC,
        endpoint: 0,
        property: "name"
    }
    const titleAff = getAffordanceFromVid(node,nameVid,0)
    let prop = tdi.AddProperty(titleAff?.name||WoTTitle, "Device name",
        "Custom device name/title used as the TD title",  WoTDataTypeString, vocab.PropDeviceTitle);
    prop.readOnly = false

    // force device location to show up
    const locationVid = {
        commandClass: titleCC,
        endpoint: 0,
        property: "location"
    }
    const locationAff = getAffordanceFromVid(node,locationVid,0)
    prop = tdi.AddProperty(locationAff?.name||"location", "Device location",
        "Description of the device location",  WoTDataTypeString,vocab.PropLocation);
    prop.readOnly = false

    // now continue with the zwave provided vids
    const vids = node.getDefinedValueIDs()
    for (const vid of vids) {
        const va = getAffordanceFromVid(node, vid, maxNrScenes)

        // let pt = getPropType(node, vid)
        if (va) {
            logVid(vidLogFD, node, vid, va)
        }

        // the vid is either config, attr, action or event based on CC
        switch (va?.affType) {
            case "action":
                // actuators accept input actions
                addAction(tdi, node, vid, va)
                break;
            case "event":
                // sensors emit events
                addEvent(tdi, node, vid, va)
                break;
            // these are varieties of properties
            case "property":
                addProperty(tdi, node, vid, va)
                break;
            default:
                //console.log("Not adding property", va?.name)
            // ignore this vid
        }
    }
    return tdi;
}


// Update the given data schema with vid data for strings, number, boolean, ...
// - title
// - description
// - readOnly, writeOnly (if defined)
// - data type: boolean, number, or string 
//   - boolean: default
//   - number: minimum, maximum, unit, enum, default
//   - string: minLength, maxLength, default
//   - default: default
function SetDataSchema(ds: DataSchema | undefined, node: ZWaveNode, vid: TranslatedValueID) {
    if (!ds) {
        return
    }
    const vidMeta = node.getValueMetadata(vid)
    ds.title = vidMeta.label ? vidMeta.label : vid.propertyName
    if (vid.endpoint) {
        ds.title += " - ("+vid.endpoint+")"
    }
    // let value = node.getValue(vid)
    // let valueName = value != undefined ? String(value) : undefined

    if (!vidMeta.readable) {
        ds.readOnly = false
        ds.writeOnly = true  // action
    } else if (!vidMeta.writeable) {
        ds.readOnly = true   // attribute or event
        ds.writeOnly = false
    } else {
        ds.readOnly = false   // config
        ds.writeOnly = false
    }
    // get more details on this property using its metadata and command class(es)
    switch (vidMeta.type) {
        case "string": {
            ds.type = WoTDataTypeString
            const vms = vidMeta as ValueMetadataString;
            ds.minLength = vms.minLength;
            ds.maxLength = vms.maxLength;
            ds.default = vms.default;
        }
            break;
        case "boolean": {
            ds.type = WoTDataTypeBool
            const vmb = vidMeta as ValueMetadataBoolean;
            ds.default = vmb.default?.toString() || undefined;
        }
            break;
        case "duration":
        case "number": {
            ds.type = WoTDataTypeNumber
            const vmn = vidMeta as ValueMetadataNumeric;
            ds.minimum = vmn.min;
            ds.maximum = vmn.max;
            // prop.steps = vmn.steps;
            ds.unit = vmn.unit;
            ds.default = vmn.default?.toString() || undefined;

            // if a list of states exist then the number is an enum.
            // convert the enum and use strings instead of numeric values
            // See also NodeValues for sending the string value
            if (vmn.states && Object.keys(vmn.states).length > 0) {
                ds.type = WoTDataTypeString
                // valueName = vmn.states[value as number]
                // // eg Operating Voltage has a value of 110 while the map has 120, 240
                // if (valueName == undefined) {
                //     valueName = String(value)
                // }
                // ds.initialValue = valueName
                // prop.allowManualEntry = (vmeta as ConfigurationMetadata).allowManualEntry || false
                ds.enum = []
                for (const k in vmn.states) {
                    ds.enum.push(vmn.states[k])
                }
                ds.default = vmn.states[vmn.default as number]
            }

        }
            break;
        case "color": {
            ds.type = WoTDataTypeNumber
        }
            break;
        case "buffer":
        case "boolean[]":
        case "number[]":
        case "string[]": {
            ds.type = WoTDataTypeArray
        }
            break;
        default: {
            // TBD: does this mean there is no schema, eg no data, eg not a value?
            ds.type = WoTDataTypeNone
        }
    }
    // ds.initialValue = valueName
    if (vidMeta.description) {
        ds.description = `${vid.commandClassName}: ${vidMeta.description}`
    } else if (vid.commandClass == CommandClasses.Configuration) {
        ds.description = `${vid.commandClassName}: ${vid.property} - ${vidMeta.label}`
    } else {
        ds.description = `${vid.commandClassName}: ${vidMeta.label}`
    }

    if (vid.propertyKey) {
        // this is a nested property
    }
    // what can we do with ccSpecific?
    if (vidMeta.ccSpecific) {
        // let addVal: unknown
        // switch (vid.commandClass) {
        //     case CommandClasses["Alarm Sensor"]:
        //         addVal = vidMeta.ccSpecific.sensorType;
        //         break;
        //     case CommandClasses["Binary Sensor"]:
        //         addVal = vidMeta.ccSpecific.sensorType;
        //         break;
        //     case CommandClasses["Indicator"]:
        //         addVal = vidMeta.ccSpecific.indicatorID;
        //         addVal = vidMeta.ccSpecific.propertyId;
        //         break;
        //     case CommandClasses["Meter"]:
        //         addVal = vidMeta.ccSpecific.meterType;
        //         addVal = vidMeta.ccSpecific.rateType;
        //         addVal = vidMeta.ccSpecific.scale;
        //         break;
        //     case CommandClasses["Multilevel Sensor"]:
        //         addVal = vidMeta.ccSpecific.sensorType;
        //         addVal = vidMeta.ccSpecific.scale;
        //         break;
        //     case CommandClasses["Multilevel Switch"]:
        //         addVal = vidMeta.ccSpecific.switchType;
        //         break;
        //     case CommandClasses["Notification"]:
        //         addVal = vidMeta.ccSpecific.notificationType;
        //         break;
        //     case CommandClasses["Thermostat Setpoint"]:
        //         addVal = vidMeta.ccSpecific.setpointType;
        //         break;
        // }
        // not sure what to do with this so add it to the description
        ds.description += "; ccSpecific=" + JSON.stringify(vidMeta.ccSpecific)
    }
}
