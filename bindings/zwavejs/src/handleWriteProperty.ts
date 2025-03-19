// ZWaveJSBinding.ts holds the entry point to the zwave binding along with its configuration
import {ZWaveNode} from "zwave-js";

import type IAgentConnection from "../hivelib/messaging/IAgentConnection.ts";
import ZWAPI, {getVidValue} from "./ZWAPI.ts";
import setValue from "./setValue.ts";
import {
    RequestMessage,
    ResponseMessage,
    StatusCompleted,
} from "../hivelib/messaging/Messages.ts";
import getLogger from "./getLogger.ts";
import {getVidFromPropertyName} from "./getAffordanceFromVid.ts";

const log = getLogger()



// handle configuration write request as defined in the TD
// @param msg is the incoming message with the 'key' containing the property to set
//
// Unlike actions, property write does not return an ActionStatus update. It either
// completed or failed.
// This returns a nil response when still running.
export function handleWriteProperty(
    req: RequestMessage, node: ZWaveNode, zwapi: ZWAPI, hc: IAgentConnection):  ResponseMessage|null {

    let err: Error | undefined
    let resp: ResponseMessage|null = null

    log.info("handleWriteProperty: node '" + node.nodeId + "' setting prop '" +
        req.name + "' to value: " + req.input)

    // FIXME: use of location CC to set name and location as per doc:
    //   https://zwave-js.github.io/node-zwave-js/#/api/node?id=name
    // This seems to be broken. Reading the location CC works but writing throws an error
    // only option for now is to set the node name/location locally.
    const propVid = getVidFromPropertyName(req.name)
    if (!propVid) {
        err = new Error("failed: unknown config: " + req.name)
        resp = req.createResponse(null,err)
    } else if (propVid?.commandClass == 119 && propVid.property == "name") {
        // note that this title change requires the TD to be republished so it shows up.
        node.name = req.input
        // this also changes the title of the TD, so resend the TD
        zwapi.onNodeUpdate(node)
        resp = req.createResponse(req.input)
    } else if (propVid?.commandClass == 119 && propVid.property == "location") {
        // TODO: use CC to set location as per doc. Doc doesn't say how though.
        node.location = req.input
        // zwapi.onValueUpdate(node, propKey, node.location)
        resp = req.createResponse(req.input)
    } else {
        // async update so return no response until completed
        setValue(node, propVid, req.input)
            .then(progress => {
                if (progress === StatusCompleted) {
                    // convert the value if this is an enum
                    const newValue = getVidValue(node, propVid)
                    resp = req.createResponse(newValue)
                    hc.sendResponse(resp)
                    // notify everyone else (no correlationID)
                    // zwapi.onValueUpdate(node, propValue, newValue)
                    zwapi.onValueUpdate(node, propVid, newValue)
                }
            })
            .catch(reqErr=>{
                let err = new Error(reqErr)
                resp = req.createResponse(null,err)
                hc.sendResponse(resp)
            })
    }
    if (err) {
        log.error(err)
    }
    return resp
}

