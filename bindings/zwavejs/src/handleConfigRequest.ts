// ZWaveJSBinding.ts holds the entry point to the zwave binding along with its configuration
import {SetValueStatus, TranslatedValueID, ValueMetadataNumeric, ZWaveNode} from "zwave-js";
import {getPropVid} from "./getPropKey";
import {ThingMessage} from "@hivelib/things/ThingMessage";
import * as tslog from 'tslog';
import {DeliveryProgress, DeliveryStatus, IHubClient} from "@hivelib/hubclient/IHubClient";
import {getEnumFromMemberName, getVidValue,  ZWAPI} from "@zwavejs/ZWAPI";
import {setValue} from "@zwavejs/setValue";

const log = new tslog.Logger()



// handle configuration requests as defined in the TD
// @param msg is the incoming message with the 'key' containing the property to set
export function handleConfigRequest(
    msg: ThingMessage, node: ZWaveNode, zwapi: ZWAPI, hc: IHubClient):  DeliveryStatus {
    let stat = new DeliveryStatus()
    let errMsg: Error | undefined

    let propKey = msg.key
    let propValue = msg.data

    stat.applied(msg)

    log.info("handleConfigRequest: node '" + node.nodeId + "' setting prop '" + propKey + "' to value: " + propValue)

    let configLower = propKey.toLowerCase()
    // FIXME: use of location CC to set name and location as per doc:
    //   https://zwave-js.github.io/node-zwave-js/#/api/node?id=name
    // This seems to be broken. Reading the location CC works but writing throws an error
    // only option for now is to set the node name/location locally.
    let propVid = getPropVid(propKey)
    if (!propVid) {
        errMsg = new Error("failed: unknown config: " + propKey)
    } else if (propVid?.commandClass == 119 && propVid.property == "name") {
        // note that this title change requires the TD to be republished so it shows up.
        node.name = propValue
        stat.completed(msg)
        // this also changes the title of the TD, so resend the TD
        zwapi.onNodeUpdate(node)
    } else if (propVid?.commandClass == 119 && propVid.property == "location") {
        // TODO: use CC to set location as per doc. Doc doesn't say how though.
        node.location = propValue
        stat.completed(msg)
        // zwapi.onValueUpdate(node, propKey, node.location)
    } else {
        // convert the value if this is an enum
        setValue(node, propVid, propValue)
            .then(stat => {
                stat.messageID = msg.messageID
                // notify the sender of the update (with messageID)
                hc.sendDeliveryUpdate(stat)
                // notify everyone else (no messageID)
                let newValue = getVidValue(node, propVid)
                zwapi.onValueUpdate(node, propValue, newValue)

                // TODO: intercept the value update and send it as a status update
            })
    }


    // delivery completed with error
    if (errMsg) {
        log.error(errMsg)
        stat.completed(msg, errMsg)
    }
    return stat
}

