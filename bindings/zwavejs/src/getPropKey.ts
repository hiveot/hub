import type {ValueID} from "@zwave-js/core";

// static map of key's to vid conversion
let  key2vidMap = new Map<string,ValueID>()


// getPropID returns the property key for identifying the property used in
// the TD property map and events.
//
//  Format: {vid.commandClass}-{vid.property}{vid.endpoint}[-{vid.propertyKey}]
//
// Used for TD properties, events, actions and for sending events
export function getPropKey(vid: ValueID): string {
    let propKey = String(vid.commandClass) + "-" + String(vid.property)

    if (vid.endpoint) {
        propKey += "-" + vid.endpoint
    } else {
        propKey += "-0"
    }
    if (vid.propertyKey != undefined) {
        propKey += "-" + String(vid.propertyKey)
    }
    key2vidMap.set(propKey, vid)
    return propKey
}

// getPropVid reconstructs a vid from the property key.
//
// This uses the previously saved map key
//
// Intended for finding the vid to update configuration or initiate an action.
//
// The property key format is CC-property-endpoint[-propertyKey]
export function getPropVid(key: string): ValueID|undefined {

    let vid = key2vidMap.get(key)
    return vid
}