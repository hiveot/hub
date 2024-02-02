import type {ValueID} from "@zwave-js/core";

// getPropID returns the property instance ID for identifying the property used in TD property map and events.
//  Format: {vid.CC}-{vid.property}[-{vid.propertyKey}][-{vid.endpoint}]
// Used for TD properties, events, actions and for sending events
export function getPropID(vid: ValueID): string {
    let propID = String(vid.commandClass) + "-" + String(vid.property)

    if (vid.propertyKey != undefined) {
        propID += "-" + String(vid.propertyKey)
    }
    if (vid.endpoint) {
        propID += "-" + vid.endpoint
    }
    return propID
}
