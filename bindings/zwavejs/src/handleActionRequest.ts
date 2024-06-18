// ZWaveJSBinding.ts holds the entry point to the zwave binding along with its configuration
import {InterviewStage, SetValueStatus, TranslatedValueID, ZWaveNode} from "zwave-js";
import {getPropVid} from "./getPropKey";
import * as vocab from "@hivelib/api/vocab/ht-vocab";
import {ActionTypeProperties} from "@hivelib/api/vocab/ht-vocab";
import {ThingMessage} from "@hivelib/things/ThingMessage";
import * as tslog from 'tslog';
import {DeliveryProgress, DeliveryStatus, IHubClient} from "@hivelib/hubclient/IHubClient";
import {stringToValue, ZWAPI} from "@zwavejs/ZWAPI";

const log = new tslog.Logger()



// handle controller actions as defined in the TD
// Normally this returns the delivery status to the caller.
// If delivery is in progress then use 'hc' to send further status updates.
export function  handleActionRequest(msg: ThingMessage, zwapi: ZWAPI, hc: IHubClient): DeliveryStatus {
    let stat = new DeliveryStatus()
    let errMsg: string = ""
    let actionLower = msg.key.toLowerCase()
    let targetNode: ZWaveNode | undefined
    let node = zwapi.getNodeByDeviceID(msg.thingID)
    if (node == undefined) {
        let errMsg = new Error("handleActionRequest: node for thingID" + msg.thingID + "does not exist")
        stat.Failed(msg, errMsg)
        log.error(errMsg)
        return stat
    }
    if (msg.key == ActionTypeProperties) {
        return handleConfigRequest(msg, node, zwapi, hc)
    }
    let payload = ""
    if (msg.data) {
        payload = Buffer.from(msg.data, "base64").toString()
    }
    log.info("action: " + msg.key + " - value: " + payload)
    // controller specific commands (see parseController)
    switch (actionLower) {
        case "begininclusion":
            zwapi.driver.controller.beginInclusion().then()
            break;
        case "stopinclusion":
            zwapi.driver.controller.stopInclusion().then()
            break;
        case "beginexclusion":
            zwapi.driver.controller.beginExclusion().then()
            break;
        case "stopexclusion":
            zwapi.driver.controller.stopExclusion().then()
            break;

        case "beginrebuildingroutes":
            zwapi.driver.controller.beginRebuildingRoutes()
            break;
        case "stoprebuildingroutes":
            zwapi.driver.controller.stopRebuildingRoutes()
            break;
        case "getnodeneighbors": // param nodeID
            targetNode = zwapi.getNodeByDeviceID(msg.thingID)
            if (targetNode) {
                zwapi.driver.controller.getNodeNeighbors(targetNode.id).then();
            }
            break;
        case "rebuildnoderoutes": // param nodeID
            targetNode = zwapi.getNodeByDeviceID(msg.thingID)
            if (targetNode) {
                zwapi.driver.controller.rebuildNodeRoutes(targetNode.id).then();
            }
            break;
        case "removefailednode": // param nodeID
            targetNode = zwapi.getNodeByDeviceID(msg.thingID)
            if (targetNode) {
                zwapi.driver.controller.removeFailedNode(targetNode.id).then();
            }
            break;
        // Special management actions that are accessible by writing configuration updates that are not VIDs
        // case PropTypes.Name.toLowerCase():  // FIXME: what is this. set name ???
        //     node.name = params;
        //     break;
        case "checklifelinehealth":
            node.checkLifelineHealth().then()
            break;
        case "refreshinfo":
            // do not use when node interview is not yet complete
            if (node.interviewStage == InterviewStage.Complete) {
                node.refreshInfo({waitForWakeup: true}).then()
            }
            break;
        case "refreshvalues":
            node.refreshValues().then()
            break;
        default:
            let found = false
            // VID based configuration and actions
            //  currently propertyIDs are also accepted.
            let propVid = getPropVid(msg.key)
            if (propVid) {
                setValue(payload, node, propVid)
                    .then(stat => {
                        stat.messageID = msg.messageID
                        // async update
                        hc.sendDeliveryUpdate(stat)
                        zwapi.onNodeUpdate(node)
                    })

                found = true
                break;
            }
            if (!found) {
                errMsg = "action '" + msg.key + "' is not a known action for thing '" + msg.thingID + "'"
            }
    }
    stat.Completed(msg)
    if (errMsg) {
        stat.error = errMsg
        log.error(errMsg)
    }
    return stat
}

// handle configuration requests as defined in the TD
function handleConfigRequest(
    msg: ThingMessage, node: ZWaveNode, zwapi: ZWAPI, hc: IHubClient):  DeliveryStatus {
    let stat = new DeliveryStatus()
    let errMsg :Error|undefined
    // FIXME: data should be utf8 encoded
    let payload = Buffer.from(msg.data,"base64").toString()
    let propMap = JSON.parse(payload)

    stat.Applied(msg)

    for( let k in propMap) {
        let v = propMap[k]
        log.info("handleConfigRequest: node '" + node.nodeId + "' setting prop '" + k + "' to value: " + v)

        let configLower = k.toLowerCase()
        if (configLower == vocab.PropDeviceTitle) {
            // note that this title change requires the TD to be republished so it shows up.
            node.name = v
            // TODO: use CC to set name as per doc. Doc doesn't say how though.
            // pretend this change comes from the zwave driver
            // send notification
            stat.Completed(msg)
            process.nextTick(() => {
                zwapi.onNodeUpdate(node)
            })
        } else if (configLower == vocab.PropLocation) {
            node.location = v
            stat.Completed(msg)
            zwapi.onNodeUpdate(node)
        } else {
            // determine the vid of this property and update it
            let propVid = getPropVid(k)
            if (!propVid) {
                errMsg = new Error("failed: unknown config: " + k)
            } else {
                setValue(v, node, propVid)
                    .then(stat => {
                        stat.messageID = msg.messageID
                        hc.sendDeliveryUpdate(stat)
                        if (stat.progress == DeliveryProgress.DeliveryCompleted) {
                            process.nextTick(() => {
                                zwapi.onNodeUpdate(node)
                            })
                        }
                    })

            }
        }
    }
    // delivery completed with error
    if (errMsg) {
        log.error(errMsg)
        stat.Completed(msg,errMsg)
    }
    return stat
}


// Set the Vid value
// This converts the given value into the right format
// @param node: node whose value to set
// @param vid: valueID parameter to set
// @param value: stringified value to set, if any
async function setValue(value: string, node: ZWaveNode, vid: TranslatedValueID): Promise<DeliveryStatus> {
    return new Promise<DeliveryStatus>( (resolve, reject) => {
        let dataToSet: unknown
        let stat = new DeliveryStatus()
        try {
            dataToSet = stringToValue(value, node, vid)
            node.setValue(vid, dataToSet)
                .then(res => {
                    switch (res.status) {
                        case SetValueStatus.Working:
                        // TODO progress updates
                        case SetValueStatus.Success:
                        case SetValueStatus.SuccessUnsupervised:
                            stat.progress = DeliveryProgress.DeliveryCompleted
                            break;
                        default:
                            stat.progress = DeliveryProgress.DeliveryCompleted
                            stat.error = res.message
                    }
                    resolve(stat)
                })
        } catch (reason) {
            log.error(`Failed setting value. Reason: ${reason}`)
            reject(reason)
        }
    })
}