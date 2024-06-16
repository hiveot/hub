// ZWaveJSBinding.ts holds the entry point to the zwave binding along with its configuration
import {InterviewStage, SetValueResult, SetValueStatus, ValueMetadataNumeric, ZWaveNode} from "zwave-js";
import {getPropKey, getPropVid} from "./getPropKey";
import * as vocab from "@hivelib/api/vocab/ht-vocab";
import {ActionTypeProperties} from "@hivelib/api/vocab/ht-vocab";
import {ThingMessage} from "@hivelib/things/ThingMessage";
import * as tslog from 'tslog';
import {DeliveryStatus, IHubClient} from "@hivelib/hubclient/IHubClient";
import {ZWAPI} from "@zwavejs/ZWAPI";
import {ValueID} from "@zwave-js/core";

const log = new tslog.Logger()



// handle controller actions as defined in the TD
// Normally this returns the delivery status to the caller.
// If delivery is in progress then use 'hc' to send further status updates.
export function  handleActionRequest(msg: ThingMessage, zwapi: ZWAPI, hc: IHubClient): DeliveryStatus {
        let stat = new DeliveryStatus()
        let errMsg:string = ""
        let actionLower = msg.key.toLowerCase()
        let targetNode: ZWaveNode | undefined
        let node = zwapi.getNodeByDeviceID(msg.thingID)
        if (node == undefined) {
            let errMsg = new Error("handleActionRequest: node for thingID" + msg.thingID + "does not exist")
            stat.Failed(msg, errMsg)
            return stat
        }
        if (msg.key == ActionTypeProperties) {
            return handleConfigRequest(msg, node, zwapi, hc)
        }
        let payload = Buffer.from(msg.data,"base64").toString()
        log.info("action: " + msg.key + " - value: " +msg.data)
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
                for (let vid of node.getDefinedValueIDs()) {
                    let propID = getPropKey(vid)
                    if (propID.toLowerCase() == actionLower) {
                        zwapi.setValue(node, vid, msg.data)
                        found = true
                        break;
                    }
                }
                if (!found) {
                    errMsg = "action '" + msg.key + "' is not a known action"
                }
        }
        stat.Completed(msg)
        stat.error = errMsg
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
            process.nextTick(()=>{
                zwapi.onNodeUpdate(node)
            })
        } else {
            // determine the vid of this property and update it
            let propVid = getPropVid(k)
            if (!propVid) {
                errMsg = new Error("failed: unknown config: " + k)
            } else {
                let newValue = getNewValueOfType(v, propVid, node)
                if (newValue === undefined) {
                    errMsg = new Error("failed: unknown value '"+v+"' for property '" + k+ "'")
                } else {
                    // TODO: clean this up once its working
                    // TODO: support 'transitionDuration', 'volume' and 'onProgress'
                    let res = node.setValue(propVid, newValue)
                    res.then((res:SetValueResult) => {
                        switch(res.status) {
                            case SetValueStatus.Working:
                                // TODO progress updates
                            case SetValueStatus.Success:
                            case SetValueStatus.SuccessUnsupervised:
                                stat.Completed(msg)
                                hc.sendDeliveryUpdate( stat)
                                // is this the right sop
                                process.nextTick(()=>{
                                    zwapi.onNodeUpdate(node)
                                })
                                break;
                            default:
                                errMsg = new Error(res.message)
                        }
                        log.info("setValue result status " + res.status + res.message ? res.message :"" )
                        // TODO: send status updated completed
                    }).catch(e => {
                        errMsg = new Error("failed: property " + k + ". Error: " + e)
                    })
                }
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

// convert the string value to a type suitable for the vid
function getNewValueOfType(v:string, vid:ValueID, node: ZWaveNode):any {
    let vidMeta = node.getValueMetadata(vid)
    if (!vidMeta) {
        return undefined
    }
    let vl = v.toLowerCase()
    switch (vidMeta.type) {
        case "string": return v;
        case "boolean":
            if (v === "" || vl ==="false" || v === "0" || vl==="disabled") {
                return false
            }
            return true
        case "number":
            let vmn = vidMeta as ValueMetadataNumeric;
            // handle enums. if they match
            if (vmn.states) {
                for (let enumKey in Object.keys(vmn.states)) {
                    let enumVal = vmn.states[enumKey].toLowerCase()
                    if (enumVal === vl) {
                        v = enumKey
                        break
                    }
                }
                // fall back to its numeric equivalent
            }
            return parseInt(v,10)
        case "duration":
        case "color":
            return parseInt(v,10)
        case "boolean[]":
        case "number[]":
        case "string[]":
            // TODO: support of arrays
            log.error("getNewValueOfType data type '"+vidMeta.type+"' is not supported")
    }
    return v
}