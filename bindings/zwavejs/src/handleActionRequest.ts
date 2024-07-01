// ZWaveJSBinding.ts holds the entry point to the ZWave binding along with its configuration
import {InterviewStage,  ZWaveNode} from "zwave-js";
import {getPropVid} from "./getPropKey";
import {ThingMessage} from "@hivelib/things/ThingMessage";
import * as tslog from 'tslog';
import { DeliveryStatus, IHubClient} from "@hivelib/hubclient/IHubClient";
import {getVidValue, ZWAPI} from "@zwavejs/ZWAPI";
import {MessageTypeProperty} from "@hivelib/api/vocab/ht-vocab";
import {handleConfigRequest} from "@zwavejs/handleConfigRequest";
import {setValue} from "@zwavejs/setValue";

const log = new tslog.Logger()



// handle controller actions as defined in the TD
// Normally this returns the delivery status to the caller.
// If delivery is in progress then use 'hc' to send further status updates.
export function  handleActionRequest(
    msg: ThingMessage, zwapi: ZWAPI, hc: IHubClient): DeliveryStatus {

    let stat = new DeliveryStatus()
    let errMsg: string = ""
    let actionLower = msg.key.toLowerCase()
    let targetNode: ZWaveNode | undefined
    let node = zwapi.getNodeByDeviceID(msg.thingID)
    if (node == undefined) {
        let errMsg = new Error("handleActionRequest: node for thingID" + msg.thingID + "does not exist")
        stat.failed(msg, errMsg)
        log.error(errMsg)
        return stat
    }
    if (msg.messageType == MessageTypeProperty) {
        return handleConfigRequest(msg, node, zwapi, hc)
    }
    // unmarshal the payload
    // FIXME: who does the (un)marshalling? If the form defines the
    //  contentCoding and contentType then
    // is this up to the protocol client? (as multiple protocols can be supported)
    // argument encoding?
    // response encoding? hiveot wraps the reply in a delivery status message with a reply field
    // what happens when consumer and agent use different protocol encodings?


    let actionValue = msg.data
    log.info("action: " + msg.key + " - value: " + msg.data)
    // be optimistic :)
    stat.completed(msg)
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
        case "ping":
            stat.applied(msg)
            let startTime = performance.now()
            node.ping().then((success:boolean)=>{
                let endTime = performance.now()
                let msec = Math.round(endTime-startTime)
                stat.completed(msg)
                stat.reply = (msec).toString()
                log.info("ping: "+msec+" msec")
                hc.sendDeliveryUpdate(stat)
            })
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
                setValue( node, propVid, actionValue)
                    .then(stat => {
                        stat.messageID = msg.messageID
                        // async update
                        hc.sendDeliveryUpdate(stat)
                        let newValue = getVidValue(node, propVid)
                        zwapi.onValueUpdate(node, propVid, newValue)
                    })

                found = true
                break;
            }
            if (!found) {
                errMsg = "action '" + msg.key + "' is not a known action for thing '" + msg.thingID + "'"
            }
    }
    if (errMsg) {
        stat.error = errMsg
        log.error(errMsg)
    }
    return stat
}
