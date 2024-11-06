import {
    NodeStatus,
    NodeType,
    TranslatedValueID,
    ValueMetadataBoolean,
    ValueMetadataNumeric,
    ValueMetadataString,
    ZWaveNode,
    ZWavePlusNodeType,
    ZWavePlusRoleType
} from "zwave-js";
import {CommandClasses, InterviewStage} from '@zwave-js/core';
import {ActionAffordance, EventAffordance, PropertyAffordance, TD} from "@hivelib/things/TD";
import * as vocab from "@hivelib/api/vocab/vocab";
import type {ZWAPI} from "./ZWAPI";
import {logVid} from "./logVid";
import {getPropName} from "./getPropName";
import {getVidAffordance, VidAffordance} from "./getVidAffordance";
import {getDeviceType} from "./getDeviceType";
import {DataSchema} from "@hivelib/things/dataSchema";
import {
    WoTDataTypeAnyURI,
    WoTDataTypeArray,
    WoTDataTypeBool,
    WoTDataTypeNone,
    WoTDataTypeNumber,
    WoTDataTypeString
} from "@hivelib/api/vocab/vocab";


// Add the ZWave value data to the TD as an action
function addAction(td: TD, node: ZWaveNode, vid: TranslatedValueID, name: string, va: VidAffordance): ActionAffordance {
    // let vidMeta = node.getValueMetadata(vid)

    // actions without input have no schema. How to identify these?
    let schema = new DataSchema()
    SetDataSchema(schema, node, vid)
    let action = td.AddAction(
        name, schema.title || name, schema.description, schema)
        .setVocabType(va.atType)

    if (action.input) {
        // The VID title, description belongs to the action, not the schema
        action.input.title = undefined
        action.input.description = undefined
        action.input.readOnly = false
    }
    return action
}

// Add the ZWave value data to the TD as an attribute property
function addAttribute(td: TD, node: ZWaveNode, vid: TranslatedValueID, name: string, va: VidAffordance): PropertyAffordance {

    let prop = td.AddProperty(name, va?.atType, WoTDataTypeNone, "")
    // SetDataSchema also sets the title and data type
    SetDataSchema(prop, node, vid)
    return prop
}

// Add the ZWave VID to the TD as a configuration property
function addConfig(td: TD, node: ZWaveNode, vid: TranslatedValueID, name: string, va: VidAffordance): PropertyAffordance {
    let prop = td.AddProperty(name, va.atType, WoTDataTypeNone, "")
    prop.readOnly = false
    // SetDataSchema also sets the title and data type
    SetDataSchema(prop, node, vid)

    return prop

}

// Add the ZWave VID to the TD as an event
function addEvent(td: TD, node: ZWaveNode, vid: TranslatedValueID, name: string, va: VidAffordance): EventAffordance {

    let schema = new DataSchema()
    SetDataSchema(schema, node, vid)

    let ev = td.AddEvent(name, schema.title || name, schema.description, schema)
        .setVocabType(va.atType)

        // SetDataSchema use the dataschema title, not the event data title
    if (ev.data) {
        ev.data.title = undefined
        ev.data.description = undefined
    }
    return ev

}

// parseNodeInfo converts a ZWave Node into a WoT TD document
// - extract available node attributes and configuration
// - convert ZWave vocabulary to WoT/HiveOT vocabulary
// - build a TD document containing properties, events and actions
// - if this is the controller node, add controller attributes and actions
// @param zwapi: wrapper around the zwave driver
// @param node: the zwave node definition
// @param vidLogFD: optional file handle to log VID info to CSV for further analysis
// @param maxNrScenes: limit the nr of scenes in the TD as it would bloat the TD to a massive size.
export function getNodeTD(zwapi: ZWAPI, node: ZWaveNode, vidLogFD: number | undefined, maxNrScenes: number): TD {
    let td: TD;

    //--- Step 1: TD definition
    let deviceID = zwapi.getDeviceID(node.id)
    let deviceType = getDeviceType(node)
    let title = node.name
    if (!title) {
        title = node.label || deviceID
        if (node.deviceConfig?.description) {
            title += " " + node.deviceConfig?.description
        }
    }
    let description = (node.label || deviceID) + ", " + deviceType
    if (node.deviceConfig) {
        description = node.deviceConfig.manufacturer + " " + description + ", " + node.deviceConfig.description
    }

    // if (node.deviceConfig) {
    //     description = node.deviceConfig.description
    // }
    td = new TD(deviceID, deviceType, title, description);

    //--- Step 2: Add read-only attributes that are common to many nodes
    // since none of these have standard property names, use the ZWave name instead.
    // these names must match those used in parseNodeValues()
    td.AddProperty("associationCount", "Association Count","",WoTDataTypeNumber, );

    td.AddPropertyIf(node.canSleep, "canSleep","Can Sleep",
        "Device sleeps to conserve battery",WoTDataTypeBool);
    if (node.deviceClass) {
        td.AddPropertyIf( node.deviceClass.basic,"deviceClassBasic",
            node.deviceClass.basic.label, "", WoTDataTypeString);
        td.AddPropertyIf(node.deviceClass.generic,"deviceClassGeneric",
            node.deviceClass.generic.label,"", WoTDataTypeString);
        td.AddPropertyIf(node.deviceClass.specific.label, "deviceClassSpecific",
            node.deviceClass.specific.label,"", WoTDataTypeString);
        // this.setIf("supportedCCs", node.deviceClass.generic.supportedCCs);
    }
    td.AddPropertyIf(node.deviceDatabaseUrl, "deviceDatabaseURL","Database URL",
        "Link to database with device information", WoTDataTypeString,);
    td.AddProperty(vocab.PropDeviceDescription,"Description",
        "", WoTDataTypeString)
        .setVocabType(vocab.PropDeviceDescription);
    td.AddProperty("endpointCount", "Endpoints",
        "Number of endpoints in this node",WoTDataTypeNumber);
    td.AddProperty(vocab.PropDeviceFirmwareVersion,
        "Device firmware version","", WoTDataTypeString, vocab.PropDeviceFirmwareVersion);

    td.AddPropertyIf(node.getHighestSecurityClass(), "highestSecurityClass",
        "Security Class", "",WoTDataTypeString);
    td.AddPropertyIf(node.interviewAttempts, "interviewAttempts",
        "Nr interview attempts","",WoTDataTypeNumber);
    if (node.interviewStage) {
        td.AddProperty("interviewStage", "Device Interview Stage", "",
            WoTDataTypeString).SetAsEnum(InterviewStage)
    }
    td.AddProperty("isListening", "Is Listening",
        "The device is always listening and does not sleep", WoTDataTypeBool);
    td.AddPropertyIf(node.isSecure, "isSecure","Secured",
        "Device communicates securely with controller",WoTDataTypeBool );
    td.AddPropertyIf(node.isRouting, "isRouting","Routing Device",
        "Device support message routing/forwarding (if listening)", WoTDataTypeBool );
    td.AddPropertyIf(node.isControllerNode, "isControllerNode",
        "Controller Node", "Device is a ZWave controller", WoTDataTypeBool);
    td.AddPropertyIf(node.keepAwake, "keepAwake","Keep Awake",
        "Device stays awake a bit longer before sending it to sleep",WoTDataTypeBool)
    td.AddPropertyIf(node.label, "nodeLabel","Node Label",
        "Manufacturer device label",  WoTDataTypeString,vocab.PropDeviceModel);
    td.AddProperty("lastSeen","Last Seen",
        "Time this node was last seen", WoTDataTypeString)


    td.AddPropertyIf(node.manufacturerId, "manufacturerId","Manufacturer ID",
        "", WoTDataTypeString);

    td.AddPropertyIf(node.deviceConfig?.manufacturer, "manufacturerName",
        "Manufacturer Name","", WoTDataTypeString,vocab.PropDeviceMake);

    td.AddPropertyIf(node.maxDataRate, "maxDataRate","Max data rate",
        "Device maximum communication data rate", WoTDataTypeNumber);
    if (node.nodeType) {
        // td.AddProperty("nodeType", "ZWave node type",
        //     "",WoTDataTypeNumber)
        // td.AddProperty("nodeTypeName", "ZWave node type name",
        //     "",WoTDataTypeString).SetAsEnum(NodeType)
    }
    td.AddPropertyIf(node.productId, "productId","Product ID",
        "", WoTDataTypeNumber);
    td.AddPropertyIf(node.productType, "productType","Product Type",
        "", WoTDataTypeNumber);
    td.AddPropertyIf(node.protocolVersion, "protocolVersion","ZWave protocol version",
        "", WoTDataTypeString);

    td.AddPropertyIf(node.sdkVersion, vocab.PropDeviceSoftwareVersion,"SDK version",
        "", WoTDataTypeString,vocab.PropDeviceSoftwareVersion);
    if (node.status) {
        td.AddProperty(vocab.PropDeviceStatus,"Node status",
            "", WoTDataTypeNumber, vocab.PropDeviceStatus)
            .SetAsEnum(NodeStatus)
    }
    td.AddPropertyIf(node.supportedDataRates, "supportedDataRates","ZWave Data Speed",
        "", WoTDataTypeString);

    td.AddPropertyIf(node.userIcon, "userIcon","Icon",
        "", WoTDataTypeString);

    // show whether this is ZWave+
    if (node.zwavePlusNodeType) {
        td.AddProperty("zwavePlusNodeType", "ZWave+ Node Type",
            "", WoTDataTypeNumber)
        let prop = td.AddProperty("zwavePlusNodeTypeName", "ZWave+ Node Type Name",
            "", WoTDataTypeString)
        if (node.zwavePlusNodeType != undefined) {
            prop.SetAsEnum(ZWavePlusNodeType)
        } else {
            prop.description = "Z-Wave+ Command Class is not supported"
        }
    }
    if (node.zwavePlusRoleType) {
        td.AddProperty("zwavePlusRoleType","ZWave+ Role Type",
            "",  WoTDataTypeNumber)
        td.AddProperty("zwavePlusRoleTypeName","ZWave+ Role Type Name",
            "", WoTDataTypeString)
            .SetAsEnum(ZWavePlusRoleType)
    }
    td.AddPropertyIf(node.zwavePlusVersion, "zwavePlusVersion","Z-Wave+ Version",
        "",WoTDataTypeNumber);

    // actions

    td.AddAction("checkLifelineHealth", "Check connection health",
        "Initiates tests to check the health of the connection between the controller and this node and returns the results. " +
        "This should NOT be done while there is a lot of traffic on the network because it will negatively impact the test results")

    td.AddAction("ping", "Ping", "Ping the device").output=new DataSchema({
        "title": "Duration",
        "type": WoTDataTypeNumber,
        "unit": "msec"
    })
    // todo: what type of response is expected: latency in msec

    td.AddAction("refreshInfo",  "Refresh Device Info",
        "Resets (almost) all information about this node and forces a fresh interview. " +
        "Ignored when interview is in progress. After this action, the node will "+
        "no longer be ready. This can take a long time.")
    // todo: what type of response is expected: progress status updates until completed


    td.AddAction("refreshValues", "Refresh Device Values",
        "Refresh all non-static sensor and actuator values. " +
        "Use sparingly. This can take a long time and generate a lot of traffic.")
    // todo: what type of response is expected: progress status updates until completed


    //--- Step 4: add properties, events, and actions from the ValueIDs

    //--- FIXME: use the CC for device title and location
    // Currently the name and location VID (CC 119) is only included when a
    // name and location is set, so create the Vid here manually to ensure it is
    // always there.
    // Reading this CC 119 does work and returns the value set with node.name and
    // node.location.
    let titleCC = CommandClasses["Node Naming and Location"] // 119
    let nameVid = {
        commandClass: titleCC,
        endpoint: 0,
        property: "name"
    }
    let titleKey = getPropName(nameVid)
    let prop = td.AddProperty(titleKey,"Device name",
        "Custom device name/title",  WoTDataTypeString, vocab.PropDeviceTitle);
    prop.readOnly = false

    let locationVid = {
        commandClass: titleCC,
        endpoint: 0,
        property: "location"
    }
    let locationKey = getPropName(locationVid)
    prop = td.AddProperty(locationKey,"Device location",
        "Description of the device location",  WoTDataTypeString,vocab.PropLocation);
    prop.readOnly = false

    // now continue with the other vids
    let vids = node.getDefinedValueIDs()
    for (let vid of vids) {
        let va = getVidAffordance(node, vid, maxNrScenes)

        // let pt = getPropType(node, vid)
        let tdPropName = getPropName(vid)
        if (va) {
            logVid(vidLogFD, node, vid, tdPropName, va)
        }

        // the vid is either config, attr, action or event based on CC
        switch (va?.messageType) {
            case "action":
                addAction(td, node, vid, tdPropName, va)
                break;
            case "event":
                addEvent(td, node, vid, tdPropName, va)
                break;
            case "config":
                addConfig(td, node, vid, tdPropName, va)
                break;
            case "attr":
                addAttribute(td, node, vid, tdPropName, va)
                break;
            default:
            // ignore this vid
        }
    }
    return td;
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
    let vidMeta = node.getValueMetadata(vid)
    ds.title = vidMeta.label ? vidMeta.label : vid.propertyName
    if (vid.endpoint) {
        ds.title += " - ("+vid.endpoint+")"
    }
    // let value = node.getValue(vid)
    // let valueName = value != undefined ? String(value) : undefined

    if (!vidMeta.readable) {
        ds.readOnly = false
        ds.writeOnly = true  // action
    }
    if (!vidMeta.writeable) {
        ds.readOnly = true   // attribute or event
    }
    // get more details on this property using its metadata and command class(es)
    switch (vidMeta.type) {
        case "string": {
            ds.type = WoTDataTypeString
            let vms = vidMeta as ValueMetadataString;
            ds.minLength = vms.minLength;
            ds.maxLength = vms.maxLength;
            ds.default = vms.default;
        }
            break;
        case "boolean": {
            ds.type = WoTDataTypeBool
            let vmb = vidMeta as ValueMetadataBoolean;
            ds.default = vmb.default?.toString() || undefined;
        }
            break;
        case "duration":
        case "number": {
            ds.type = WoTDataTypeNumber
            let vmn = vidMeta as ValueMetadataNumeric;
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
                ds.default = vmn.states[vmn.default]
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

//
// // Split the deviceID into homeID and nodeID
// export function splitDeviceID(deviceID: string): [string, number | undefined] {
//     let parts = deviceID.split(".")
//     if (parts.length == 2) {
//         return [parts[0], parseInt(parts[1])]
//     }
//     return ["", undefined]
// }
