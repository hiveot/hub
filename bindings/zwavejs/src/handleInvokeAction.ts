// ZWaveJSBinding.ts holds the entry point to the ZWave binding along with its configuration
import {InterviewStage,  ZWaveNode} from "zwave-js";
import {getPropVid} from "./getPropName";
import {ThingMessage} from "@hivelib/things/ThingMessage";
import * as tslog from 'tslog';
import { IAgentClient} from "@hivelib/hubclient/IAgentClient";
import {getVidValue, ZWAPI} from "@zwavejs/ZWAPI";
import {HTOpUpdateProperty} from "@hivelib/api/vocab/vocab.js";
import {handleWriteProperty} from "@zwavejs/handleWriteProperty";
import {setValue} from "@zwavejs/setValue";
import {ActionStatus} from "@hivelib/hubclient/ActionStatus";

const log = new tslog.Logger()



// handle controller actions as defined in the TD
// Normally this returns the delivery status to the caller.
// If delivery is in progress then use 'hc' to send further status updates.
export function  handleInvokeAction(
    msg: ThingMessage, zwapi: ZWAPI, hc: IAgentClient): ActionStatus {

    let stat = new ActionStatus()
    let errMsg: string = ""
    let actionLower = msg.name.toLowerCase()
    let targetNode: ZWaveNode | undefined
    let node = zwapi.getNodeByDeviceID(msg.thingID)
    if (node == undefined) {
        let errMsg = new Error("handleActionRequest: node for thingID" + msg.thingID + "does not exist")
        stat.failed(msg, errMsg)
        log.error(errMsg)
        return stat
    }
    if (msg.operation == HTOpUpdateProperty) {
        return handleWriteProperty(msg, node, zwapi, hc)
    }
    // unmarshal the payload
    // FIXME: who does the (un)marshalling? If the form defines the
    //  contentCoding and contentType then
    // is this up to the protocol client? (as multiple protocols can be supported)
    // argument encoding?
    // response encoding? hiveot wraps the reply in a delivery status message with a reply field
    // what happens when consumer and agent use different protocol encodings?


    let actionValue = msg.data
    log.info("action: " + msg.name + " - value: " + msg.data)
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
            stat.delivered(msg)
            let startTime = performance.now()
            node.ping().then((success:boolean)=>{
                let endTime = performance.now()
                let msec = Math.round(endTime-startTime)
                stat.completed(msg, msec)
                log.info("ping '"+msg.thingID+"': "+msec+" msec")
                hc.pubProgressUpdate(stat)
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
            // FIXME: convert actionValue to expected type
            let propVid = getPropVid(msg.name)
            if (propVid) {
                setValue(node, propVid, actionValue)
                    .then(stat => {
                        stat.requestID = msg.requestID
                        // async update
                        hc.pubProgressUpdate(stat)
                        let newValue = getVidValue(node, propVid)
                        zwapi.onValueUpdate(node, propVid, newValue)
                    })
                    .catch(err => {
                        errMsg = err.toString()
                    })
                found = true
                break;
            }
            if (!found) {
                errMsg = "action '" + msg.name + "' is not a known action for thing '" + msg.thingID + "'"
            }
    }
    if (errMsg) {
        stat.error = errMsg
        log.error(errMsg)
    }
    return stat
}
