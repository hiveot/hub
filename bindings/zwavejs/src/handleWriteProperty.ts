// ZWaveJSBinding.ts holds the entry point to the zwave binding along with its configuration
import {ZWaveNode} from "zwave-js";
import {getPropVid} from "./getPropName";
import {IAgentConnection} from "@hivelib/transports/IAgentConnection";
import {getEnumFromMemberName, getVidValue,  ZWAPI} from "@zwavejs/ZWAPI";
import {setValue} from "@zwavejs/setValue";
import {RequestMessage, ResponseMessage} from "@hivelib/transports/Messages";
import {getlogger} from "@zwavejs/getLogger";

const log = getlogger()



// handle configuration write request as defined in the TD
// @param msg is the incoming message with the 'key' containing the property to set
export function handleWriteProperty(
    req: RequestMessage, node: ZWaveNode, zwapi: ZWAPI, hc: IAgentConnection):  ResponseMessage {

    let err: Error | undefined

    let propKey = req.name
    let propValue = req.input

    log.info("handleConfigRequest: node '" + node.nodeId + "' setting prop '" + propKey + "' to value: " + propValue)

    // FIXME: use of location CC to set name and location as per doc:
    //   https://zwave-js.github.io/node-zwave-js/#/api/node?id=name
    // This seems to be broken. Reading the location CC works but writing throws an error
    // only option for now is to set the node name/location locally.
    let propVid = getPropVid(propKey)
    if (!propVid) {
        err = new Error("failed: unknown config: " + propKey)
    } else if (propVid?.commandClass == 119 && propVid.property == "name") {
        // note that this title change requires the TD to be republished so it shows up.
        node.name = propValue
        // this also changes the title of the TD, so resend the TD
        zwapi.onNodeUpdate(node)
    } else if (propVid?.commandClass == 119 && propVid.property == "location") {
        // TODO: use CC to set location as per doc. Doc doesn't say how though.
        node.location = propValue
        // zwapi.onValueUpdate(node, propKey, node.location)
    } else {
        // convert the value if this is an enum
        // async update
        setValue(node, propVid, propValue)
            .then(progress => {
                let newValue = getVidValue(node, propVid)

                let resp = req.createResponse(newValue)
                resp.status=progress
                // notify the sender of the update (with correlationID)
                hc.sendResponse(resp)
                // notify everyone else (no correlationID)
                // zwapi.onValueUpdate(node, propValue, newValue)
                zwapi.onValueUpdate(node, propVid, newValue)
            })
            .catch(reqerr=>{
                err = new Error(reqerr)
            })
    }

    // delivery completed with error
    if (err) {
        log.error(err)
    }
    return req.createResponse(null,err)
}

