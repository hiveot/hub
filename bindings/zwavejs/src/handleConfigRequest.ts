// ZWaveJSBinding.ts holds the entry point to the zwave binding along with its configuration
import {InterviewStage, SetValueStatus, TranslatedValueID, ZWaveNode} from "zwave-js";
import {getPropVid} from "./getPropKey";
import * as vocab from "@hivelib/api/vocab/ht-vocab";
import {ThingMessage} from "@hivelib/things/ThingMessage";
import * as tslog from 'tslog';
import {DeliveryProgress, DeliveryStatus, IHubClient} from "@hivelib/hubclient/IHubClient";
import {stringToValue, ZWAPI} from "@zwavejs/ZWAPI";
import {MessageTypeProperty} from "@hivelib/api/vocab/ht-vocab";

const log = new tslog.Logger()



// handle configuration requests as defined in the TD
// @param msg is the incoming message with the 'key' containing the property to set
export function handleConfigRequest(
    msg: ThingMessage, node: ZWaveNode, zwapi: ZWAPI, hc: IHubClient):  DeliveryStatus {
    let stat = new DeliveryStatus()
    let errMsg: Error | undefined

    let propKey = msg.key
    let propValue = msg.data

    stat.Applied(msg)

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
        stat.Completed(msg)
        // this also changes the title of the TD, so resend the TD
        zwapi.onNodeUpdate(node)
    } else if (propVid?.commandClass == 119 && propVid.property == "location") {
        // TODO: use CC to set location as per doc. Doc doesn't say how though.
        node.location = propValue
        stat.Completed(msg)
        // zwapi.onValueUpdate(node, propKey, node.location)
    } else {
        setValue(node, propVid, propValue)
            .then(stat => {
                stat.messageID = msg.messageID
                // notify the sender of the update (with messageID)
                hc.sendDeliveryUpdate(stat)
                // notify everyone else (no messageID)
                let newValue = node.getValue(propVid)
                zwapi.onValueUpdate(node, propValue, newValue)

                // TODO: intercept the value update and send it as a status update
            })
    }


    // delivery completed with error
    if (errMsg) {
        log.error(errMsg)
        stat.Completed(msg, errMsg)
    }
    return stat
}


// Set a new Vid value.
// This converts the given value into the right format
// @param node: node whose value to set
// @param vid: valueID parameter to set
// @param value: stringified value to set, if any
// this returns a delivery status for returning to the hub
export async function setValue(node: ZWaveNode, vid: TranslatedValueID, value: string): Promise<DeliveryStatus> {
    return new Promise<DeliveryStatus>( (resolve, reject) => {
        let dataToSet: unknown
        let stat = new DeliveryStatus()
        try {
            dataToSet = stringToValue(value, node, vid)
            node.setValue(vid, dataToSet, {onProgress:(prog)=>{
                // 0: queued
                // 1: active (currently behing handled)
                // 2: completed
                // 3: failed
                // log.info("setValue progress: ", prog.state.toString())
            }})
                .then(res => {
                    switch (res.status) {
                        case SetValueStatus.Working:
                            stat.progress = DeliveryProgress.DeliveryApplied
                            break;
                        // TODO progress updates
                        case SetValueStatus.Success:
                        case SetValueStatus.SuccessUnsupervised:
                            stat.progress = DeliveryProgress.DeliveryCompleted
                            break;
                        case SetValueStatus.EndpointNotFound:
                        case SetValueStatus.NotImplemented:
                            stat.progress = DeliveryProgress.DeliveryFailed
                            stat.error = res.message
                            break;
                        case SetValueStatus.InvalidValue:
                            stat.progress = DeliveryProgress.DeliveryCompleted
                            stat.error = res.message
                            break
                        default:
                            stat.progress = DeliveryProgress.DeliveryApplied
                    }
                    stat.error = res.message
                    resolve(stat)
                })
        } catch (reason) {
            log.error(`Failed setting value. Reason: ${reason}`)
            reject(reason)
        }
    })
}