import type { ZWaveNode } from "zwave-js";
import { CommandClasses, ValueID } from "@zwave-js/core";
import { ActionTypes, EventTypes } from "@hivelib/vocab/vocabulary";


// ValueID to Affordance classification
export interface VidAffordance {
    atType: string, // @type of the property, event or action, "" if not known
    affordance: "action" | "event" | "attr" | "config" | undefined
}

// Override of zwavejs VID to HiveOT action, event, config or attributes
// This allows fine-grained adjustments of the default rules for mapping VIDs.
// Key is the VID {CC}-{propertyName}[-{propertyKey}] (if propertyKey exists)
const overrideMap: Map<string, Partial<VidAffordance> | undefined> = new Map([
    // Basic 0x20 (32): ignore these VIDs as there are more specific ones
    ["32-currentValue", {}],
    ["32-targetValue", {}],
    ["32-duration", {}],
    ["32-restorePrevious", {}],

    // Binary Switch 0x25 (37) is an actuator
    ["37-currentValue", { atType: ActionTypes.Switch, affordance: "event" }],
    ["37-targetValue", { atType: ActionTypes.Switch, affordance: "action" }],

    // Multilevel Switch (38) is an actuator

    // Binary Sensor (48)
    ["48-Any", { atType: EventTypes.Alarm, affordance: "event" }],

    // Meter - electrical
    ["50-value-65537", { atType: EventTypes.Energy, affordance: "event" }],
    ["50-value-66049", { atType: EventTypes.Power, affordance: "event" }],
    ["50-value-66561", { atType: EventTypes.Voltage, affordance: "event" }],
    ["50-value-66817", { atType: EventTypes.Current, affordance: "event" }],
    ["50-reset", { affordance: "config" }], // for managers, not operators

    // Notification
    ["113-Home Security-Motion sensor status", { atType: EventTypes.Motion, affordance: "event" }],
]);


// Default rules to determine whether the vid is an attr, config, event, action or to be ignored
// this returns
//  action: the vid is writable and not readable
//  event: the vid is a readonly command CC ?
//  attr: the vid is read-only, not an event, and has a value or default
//  config: the vid is writable, not an action, and has a value or default
//  undefined if the vid CC is deprecated or the vid is to be ignored
function defaultVidAffordance(node: ZWaveNode, vid: ValueID, maxNrScenes: number): "action" | "event" | "config" | "attr" | undefined {
    let vidMeta = node.getValueMetadata(vid)

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
            return vidMeta.writeable ? "action" : "event";
        }

        //-- CC's for data reporting devices
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
            return vidMeta.writeable ? "config" : "attr"
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
            return "config"
        }

        case CommandClasses["Wake Up"]: {
            // wakeup interval is config, wakeup report is attr, wakeup notification is event
            return vidMeta.writeable ? "config" : "attr"
        }

        //--- deprecated CCs
        case CommandClasses["All Switch"]:  //
        case CommandClasses["Application Capability"]:  // obsolete
        case CommandClasses["Alarm Sensor"]:  // nodes also have Notification CC
            {
                return undefined
            }
    }

    if (!vidMeta.readable) {
        return vidMeta.writeable ? "action" : "event"
    }
    if (vidMeta.writeable) {
        return "config"
    }
    return "attr"
}


// getVidAffordance determines how to represent the Vid in the TD.
// This first uses the default rules based mainly on the Vid's CommandClass and writability,
// then applies the override map to deal with individual Vids.
// The override map is currently hard coded but intended to be moved to a configuration file.
//
// Returns a VidAffordance object or undefined if the Vid is to be ignored.
export function getVidAffordance(node: ZWaveNode, vid: ValueID, maxNrScenes: number): VidAffordance | undefined {
    // Determine default values for @type and affordance
    let affordance = defaultVidAffordance(node, vid, maxNrScenes)
    let atType = ""
    let va: VidAffordance = {
        atType: atType,
        affordance: affordance
    }

    // Apply values from an override
    let mapKey = vid.commandClass + "-" + String(vid.property)
    if (vid.propertyKey != undefined) {
        mapKey += "-" + String(vid.propertyKey)
    }
    if (overrideMap.has(mapKey)) {
        let override = overrideMap.get(mapKey)
        if (!override) {
            return undefined
        }
        if (override.atType != undefined) {
            va.atType = override.atType
        }
        if (override.affordance != undefined) {
            va.affordance = override.affordance
        }
    }
    return va.affordance ? va : undefined
}
