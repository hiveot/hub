import {ValueMetadata, ZWaveNode} from "zwave-js";
import {CommandClasses, type ValueID} from "@zwave-js/core";
import * as vocab from "../hivelib/api/vocab/vocab.js";
// import {ActionAffordance} from "../hivelib/wot/TD.ts";
// import {ValueMetadata} from "@zwave-js/core/build/values/Metadata";


// ValueID to TD event,action or property affordance type
export interface VidAffordance {
    // @type of the property, event or action, "" if not known
    atType: string,
    // The property/event/action name that identifies this vid
    name: string
    // attr is read-only property; config is writable property
    affType: "action" | "event" | "property"  | undefined

    // value metadata
    meta: ValueMetadata

    // affordance title
    title: string | undefined
    //
    vidID:string
}


// static map of action name to vid conversion
// used for invoking actions
const act2vidMap = new Map<string,ValueID>()
// static map of property name to vid conversion
// used for handling property writes
const prop2vidMap = new Map<string,ValueID>()


// Override map of zwavejs VID to HiveOT action, event, config or attributes.
//
// This overrides the vid derived message type to allow fine-grained adjustments
// of the default rules for mapping VIDs.
// Key is the VID {CC}-{propertyName}[-{propertyKey}] (if propertyKey exists)
const overrideMap: Map<string, Partial<VidAffordance> | undefined> = new Map([
    // Basic 0x20 (32): ignore these VIDs as there are more specific ones
    ["32-currentValue", {}],
    ["32-targetValue", {}],
    ["32-duration", {}],
    ["32-restorePrevious", {}],

    // Binary Switch 0x25 (37) is an actuator; convert it to a property and action with the same name
    ["37-currentValue", {atType: vocab.PropSwitch, affType: "property", name:"switch", "title":"On/Off Status"} ],
    ["37-targetValue", {atType: vocab.ActionSwitch, affType: "action", name:"switch", "title":"On/Off Control"}],

    // Multilevel Switch (38) is an actuator

    // Binary Sensor (48)
    ["48-Any", {atType: vocab.PropAlarmStatus, affType: "event"}],

    // Meter - electrical
    ["50-value-65537", {atType: vocab.PropElectricEnergy, affType: "event", name: "energy"}],
    ["50-value-66049", {atType: vocab.PropElectricPower, affType: "event", name: "power"}],
    ["50-value-66561", {atType: vocab.PropElectricVoltage, affType: "event", name: "voltage"}],
    ["50-value-66817", {atType: vocab.PropElectricCurrent, affType: "event", name: "current"}],
    // reset is an action not a property; stateless
    ["50-reset", {affType: "action"}], // for managers, not operators

    // Notification
    ["113-Home Security-Motion sensor status",
        {atType: vocab.PropAlarmMotion, affType: "event"}],
]);


// Default rules to determine whether the vid is an attr, config, event, action or
// to be ignored.
//
// @param node the node whose affordance to get
// @param vid the vid (property/event/action) whose affordance to get
// @param maxNrScenes  only provide affordances for a limited nr of scenes
//
// This returns:
//  actuator: the vid is writable and has a value or default
//  action: the vid is writable, not an actuator, and not readable
//  event: the vid is a readonly command CC and not state
//  sensor: the vid is read-only, has a value or default, and is device state
//  attr: the vid is read-only, not an event or sensor, and has a value or default
//  config: the vid is writable, not an actuator, and has a value or default
//  undefined if the vid CC is deprecated or the vid is to be ignored
function defaultVidAffordance(node: ZWaveNode, vid: ValueID, maxNrScenes: number):
       "action" | "event" |"property"| undefined {

    const vidMeta = node.getValueMetadata(vid)

    // 1. Binary Switch: targetValue is a config, not an action;
    // 1b. valueChangeOptions =["transitionDuration"]   is this the config parameter?
    // 2. Meter:reset[-1] is an action, not an event  (DSC-18103)
    // OK: Param 1: report type doesn't show => replaced by Automatic Report Group 1 - Current, Power,Voltage,kWh
    // 3. Param 255 Reset config to default is a config, not an event (DSC-18103)
    // 4. Param 254 Device Tag doesn't show (DSC-18103)
    // 5. Param 252 Enable/disable Lock Configuration doesn't show (DSC-18103)

    switch (vid.commandClass) {
        // Basic offers get,set,duration,restorePrevious which make no sense on (multi)-sensors
        // and power meters. Since other command-classes provide more specific capabilities Basic is ignored.
        case CommandClasses.Basic: {
            return undefined
        }
        //--- CC's for actions/actuator devices
        case CommandClasses["Alarm Silence"]:
        case CommandClasses["Barrier Operator"]:
        case CommandClasses["Binary Switch"]:
        case CommandClasses["Binary Toggle Switch"]:
        case CommandClasses["Door Lock"]:
        case CommandClasses["HRV Control"]:
        case CommandClasses["Humidity Control Mode"]:
        case CommandClasses["Indicator"]:
        case CommandClasses["Multilevel Switch"]:
        case CommandClasses["Simple AV Control"]:
        case CommandClasses["Window Covering"]: {
            if (vidMeta.writeable) {
                return "action";
            }
            return "property"
        }

        //-- CC's for data reporting devices (sensor)
        case CommandClasses["Authentication"]:
        case CommandClasses["Binary Sensor"]:
        case CommandClasses["Central Scene"]:
        case CommandClasses["Entry Control"]:
        case CommandClasses["Energy Production"]:
        case CommandClasses["HRV Status"]:
        case CommandClasses["Humidity Control Operating State"]:
        case CommandClasses["Multilevel Sensor"]:
        case CommandClasses.Meter:
        case CommandClasses["Meter Table Monitor"]:
        case CommandClasses.Notification:
        case CommandClasses["Sound Switch"]:
        case CommandClasses["Thermostat Fan State"]:
        case CommandClasses["Thermostat Operating State"]: {
            return "event"
        }

        //--- CC's for configuration or attributes
        case CommandClasses["Anti-Theft"]:
        case CommandClasses["Anti-Theft Unlock"]:
        case CommandClasses["Color Switch"]:
        case CommandClasses["Configuration"]:
        case CommandClasses["Generic Schedule"]:
        case CommandClasses["Humidity Control Setpoint"]:
        case CommandClasses["Irrigation"]:
        case CommandClasses["Meter Table Configuration"]:
        case CommandClasses["Meter Table Push Configuration"]:
        case CommandClasses["Schedule"]:

        case CommandClasses["Thermostat Fan Mode"]:
        case CommandClasses["Thermostat Mode"]:
        case CommandClasses["Thermostat Setpoint"]:
        case CommandClasses["Thermostat Setback"]:
        case CommandClasses["Tariff Table Configuration"]:
        case CommandClasses["User Code"]: {
            return "property";
        }

        // Reduce nr of Scene Actuator Configurations
        // ignore all scene configuration for scenes 10-255 to reduce the amount of unused properties
        // tbd: convert 255 scene's to a map?
        // Note that the DSC18103 is a binary switch while still sending level and dimming duration VIDs
        // configuration report command: CC=Scene Actuator Configuration
        //                               Command = SCENE_ACTUATOR_CONF_REPORT    <- Where can this be found?
        // TODO: ignore dimming-duration if this is 0 as per doc:
        //  "supporting actuator nodes without duration capabilities MUST ignore this found and should set it to 0"
        //  0 means instantly; 1-127 in 1-second resolution; 128-254  in 1 minute resolution (1-127 minutes)
        case CommandClasses["Scene Controller Configuration"]:   // 1..255 scene IDs
        case CommandClasses["Scene Actuator Configuration"]: {
            if (vid.property == "dimmingDuration" || vid.property == "level") {
                if (vid.propertyKey && Number(vid.propertyKey) > maxNrScenes) {
                    return undefined;
                }
            }
            return "property"
        }

        case CommandClasses["Wake Up"]: {
            // wakeup interval is config, wakeup report is attr, wakeup notification is event
            return "property"
        }

        //--- deprecated CCs
        case CommandClasses["All Switch"]:  //
        case CommandClasses["Application Capability"]:  // obsolete
        case CommandClasses["Alarm Sensor"]:  // nodes also have Notification CC
        {
            return undefined
        }
    }

    // write-only
    if (!vidMeta.readable) {
        return vidMeta.writeable ? "action" : "event"
    }
    return "property"
}


// getAffordanceFromVid determines how to represent the Vid in the TD.
//
// This first gets the vidID based on the Vid's CommandClass, zw property name and endpoint,
// Then the override map is applied  to deal with specific Vids.
// The override map is currently hard coded but intended to be moved to a configuration file.
//
// Returns a VidAffordance object or undefined if the Vid is to be ignored.
export default function getAffordanceFromVid(node: ZWaveNode, vid: ValueID, maxNrScenes: number): VidAffordance | undefined {
    // Determine default values for @type and affordance
    const affordance = defaultVidAffordance(node, vid, maxNrScenes)
    if (!affordance) {
        return
    }

    const va: VidAffordance = {
        vidID: "",
        name: "",
        atType: "",
        affType: affordance,
        title:"",
        meta: node.getValueMetadata(vid)
    }


    // Apply values from an override
    // first determine the unique vid identifier. This is used as the default property name.
    //
    //  Format: {vid.commandClass}-{vid.property}[-{vid.endpoint}][-{vid.propertyKey}]
    //  With spaces replaced by underscore '_'.
    //
    va.vidID = getVidID(vid)

    // the overrideMap can replace property info
    va.name = va.vidID
    if (overrideMap.has(va.vidID)) {
        const override = overrideMap.get(va.vidID)
        if (!override) {
            return undefined
        }
        if (override.atType) {
            va.atType = override.atType
        }
        if (override.affType) {
            va.affType = override.affType
        }
        if (override.name) {
            va.name = override.name
        }
        // the property title can be overridden
        if (override.title) {
            va.title = override.title
        }
    }

    // store the vid by name for easy lookup on incoming action or property write requests
    // since vids can change when they are refreshed, keep updating it.
    if (affordance === "action") {
        // in case actions and properties have different vid
        act2vidMap.set(va.name, vid)
    } else {
        prop2vidMap.set(va.name, vid)
    }
    // store the vid by property name for easy lookup on incoming requests
    // since vids can change when they are refreshed, keep updating it.
    // prop2vidMap.set(va.name, vid)
    return va
}


/**
    getVidID returns the unique identifier of the vid and update the vidID:vid map
    don't use directly. Use getAffordanceFromVid instead.

    Format: {vid.commandClass}-{vid.property}[-{vid.propertyKey}][-{vid.endpoint}]

    Spaces are replaced by _

    Used for TD properties, events, actions and for sending events
*/
function getVidID(vid: ValueID): string {
    let vidID = String(vid.commandClass) + "-" + String(vid.property)

    if (vid.propertyKey != undefined) {
        vidID += "-" + String(vid.propertyKey)
    }

    // only if there are multiple endpoints then append it
    if (vid.endpoint) {
        vidID += "-" + String(vid.endpoint)
    }

    // property/event/action names cannot have spaces
    vidID = vidID.replaceAll(" ", "_")
    return vidID
}


// getVidFromActionName returns the vid of an action affordance.
//
// This uses the previously saved action->vid map.
// Intended for finding the vid to update configuration or initiate an action.
//
// This returns undefined if no such property exists.
export function getVidFromActionName(name:string): ValueID|undefined {
    const vid = act2vidMap.get(name)
    return vid
}


// getVidFromPropertyName reconstructs a vid of a property affordance.
//
// This uses the previously saved property->vid map.
// Intended for finding the vid to update configuration or initiate an action.
//
// This returns undefined if no such property exists.
export function getVidFromPropertyName(name:string): ValueID|undefined {
    const vid = prop2vidMap.get(name)
    return vid
}