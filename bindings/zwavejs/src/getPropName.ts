import type {ValueID} from "npm:@zwave-js/core";

// static map of key's to vid conversion
let  key2vidMap = new Map<string,ValueID>()


// getPropName returns the property key for identifying the property used in
// the TD property map and events.
//
//  Format: {vid.commandClass}-{vid.property}{vid.endpoint}[-{vid.propertyKey}]
//
// Used for TD properties, events, actions and for sending events
export function getPropName(vid: ValueID): string {
    let propName = String(vid.commandClass) + "-" + String(vid.property)

    if (vid.endpoint) {
        propName += "-" + vid.endpoint
    } else {
        propName += "-0"
    }
    if (vid.propertyKey != undefined) {
        propName += "-" + String(vid.propertyKey)
    }
    // property/event/action names cannot have spaces
    propName = propName.replaceAll(" ", "_")
    key2vidMap.set(propName, vid)
    return propName
}

// getPropVid reconstructs a vid from the property key.
//
// This uses the previously saved map key
//
// Intended for finding the vid to update configuration or initiate an action.
//
// The property key format is CC-property-endpoint[-propertyKey]
export function getPropVid(name: string): ValueID|undefined {

    let vid = key2vidMap.get(name)
    return vid
}