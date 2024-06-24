// ZWaveJSBinding.ts holds the entry point to the zwave binding along with its configuration
import {InterviewStage, SetValueStatus, TranslatedValueID, ZWaveNode} from "zwave-js";
import {getPropVid} from "./getPropKey";
import * as vocab from "@hivelib/api/vocab/ht-vocab";
import {ThingMessage} from "@hivelib/things/ThingMessage";
import * as tslog from 'tslog';
import {DeliveryProgress, DeliveryStatus, IHubClient} from "@hivelib/hubclient/IHubClient";
import {stringToValue, ZWAPI} from "@zwavejs/ZWAPI";
import {MessageTypeProperty} from "@hivelib/api/vocab/ht-vocab";
import {handleConfigRequest, setValue} from "@zwavejs/handleConfigRequest";

const log = new tslog.Logger()



// handle controller actions as defined in the TD
// Normally this returns the delivery status to the caller.
// If delivery is in progress then use 'hc' to send further status updates.
export function  handleHubMessage(msg: ThingMessage, zwapi: ZWAPI, hc: IHubClient): DeliveryStatus {
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
    if (msg.messageType == MessageTypeProperty) {
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
            // FIXME: only allow defined actions
            let propVid = getPropVid(msg.key)
            if (propVid) {
                setValue( node, propVid, payload)
                    .then(stat => {
                        stat.messageID = msg.messageID
                        // async update
                        hc.sendDeliveryUpdate(stat)
                        let newValue = node.getValue(propVid)
                        zwapi.onValueUpdate(node, propVid, newValue)
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
