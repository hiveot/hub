// ZWaveJSBinding.ts holds the entry point to the ZWave binding along with its configuration
import {InterviewStage,  ZWaveNode} from "zwave-js";
import type IAgentConnection from "../hivelib/messaging/IAgentConnection.ts";
import {OpWriteProperty} from "../hivelib/api/vocab/vocab.js";
import {
    RequestMessage,
    ResponseMessage,
    StatusCompleted,
    StatusRunning
} from "../hivelib/messaging/Messages.ts";

import {getPropVid} from "./getPropName.ts";
import ZWAPI, {getVidValue} from "./ZWAPI.ts";
import {handleWriteProperty} from "./handleWriteProperty.ts";
import setValue from "./setValue.ts";
import getLogger from "./getLogger.ts";

const log = getLogger()



// handle controller actions as defined in the TD
// Normally this returns the delivery status to the caller.
// If delivery is in progress then use 'hc' to send further status updates.
export default function  handleRequest(
    req: RequestMessage, zwapi: ZWAPI, hc: IAgentConnection): ResponseMessage {

    let err: Error | undefined
    let output: any = null
    let status = StatusCompleted
    let resp: ResponseMessage

    const actionLower = req.name.toLowerCase()
    let targetNode: ZWaveNode | undefined
    const node = zwapi.getNodeByDeviceID(req.thingID)
    if (node == undefined) {
        const errMsg = new Error("handleActionRequest: node for thingID" + req.thingID + "does not exist")
        log.error(errMsg)
        return req.createResponse(null, errMsg)
    }
    if (req.operation == OpWriteProperty) {
        return handleWriteProperty(req, node, zwapi, hc)
    }

    const actionValue = req.input
    log.info("action: " + req.name + " - value: " + req.input)
    // be optimistic :)
    // controller specific commands (see parseController)
    switch (actionLower) {
        case "begininclusion": {
            zwapi.driver.controller.beginInclusion().then()
        } break;
        case "stopinclusion": {
            zwapi.driver.controller.stopInclusion().then()
        } break;
        case "beginexclusion": {
            zwapi.driver.controller.beginExclusion().then()
        } break;
        case "stopexclusion": {
            zwapi.driver.controller.stopExclusion().then()
        } break;
        case "beginrebuildingroutes": {
            zwapi.driver.controller.beginRebuildingRoutes()
        } break;
        case "stoprebuildingroutes": {
            zwapi.driver.controller.stopRebuildingRoutes()
        } break;
        case "getnodeneighbors": { // param nodeID
            targetNode = zwapi.getNodeByDeviceID(req.thingID)
            if (targetNode) {
                zwapi.driver.controller.getNodeNeighbors(targetNode.id).then();
            }
        } break;
        case "rebuildnoderoutes": { // param nodeID
            targetNode = zwapi.getNodeByDeviceID(req.thingID)
            if (targetNode) {
                zwapi.driver.controller.rebuildNodeRoutes(targetNode.id).then();
            }
        } break;
        case "removefailednode": { // param nodeID
            targetNode = zwapi.getNodeByDeviceID(req.thingID)
            if (targetNode) {
                zwapi.driver.controller.removeFailedNode(targetNode.id).then();
            }
        } break;

        // Special management actions that are accessible by writing configuration updates that are not VIDs
        // case PropTypes.Name.toLowerCase():  // FIXME: what is this. set name ???
        //     node.name = params;
        //     break;
        case "checklifelinehealth": {
            status = StatusRunning // async response
            // 3 runs; return rating
            node.checkLifelineHealth(3)
                .then((ev) => {
                    resp = req.createResponse(ev.rating)
                    hc.sendResponse(resp)
                })
                .catch(err => {
                    resp = req.createResponse(null, err)
                    hc.sendResponse(resp)
                });
        } break;

        case "ping": {
            status = StatusRunning // async response

            // ping a node. The 'completed' response is sent async
            const startTime = performance.now()
            node.ping().then((_success: boolean) => {
                const endTime = performance.now()
                const msec = Math.round(endTime - startTime)
                resp = req.createResponse(msec)
                log.info("ping '" + req.thingID + "': " + msec + " msec")
                hc.sendResponse(resp)
            })
        } break;

        case "refreshinfo": {
            status = StatusRunning
            // doc warning: do not call refreshInfo when node interview is not yet complete
            if (node.interviewStage == InterviewStage.Complete) {
                node.refreshInfo({waitForWakeup: true})
                    .then((result) => {
                        log.info("refreshinfo. StartedResult:", result)
                        resp = req.createResponse(null)
                        hc.sendResponse(resp) // async
                    })
                    .catch(err => {
                        log.info("refreshinfo failed: ", err)
                        resp = req.createResponse(null, err)
                        hc.sendResponse(resp) // async
                    })
            } else {
                // a previous request was still running.
                err = new Error("refreshinfo is already running")
            }
        } break;

        case "refreshvalues": {
            status = StatusRunning
            // this can take 10-20 seconds
            node.refreshValues().then((_res) => {
                resp = req.createResponse(null)
                hc.sendResponse(resp) // async
                log.info("refreshvalues completed")
            }).catch(err => {
                log.info("refreshvalues failed: ", err)
                resp = req.createResponse(null, err)
                hc.sendResponse(resp) // async
            })
        } break;

        default: {
            let found = false
            // VID based configuration and actions
            //  currently propertyIDs are also accepted.
            // FIXME: only allow defined actions
            // FIXME: convert actionValue to expected type
            const propVid = getPropVid(req.name)
            if (propVid) {
                status = StatusRunning
                setValue(node, propVid, actionValue)
                    .then(progress => {
                        const newValue = getVidValue(node, propVid)
                        resp = req.createResponse(newValue)
                        resp.status = progress
                        hc.sendResponse(resp)
                        zwapi.onValueUpdate(node, propVid, newValue)
                    })
                    .catch(err => {
                        resp = req.createResponse(null, err)
                        hc.sendResponse(resp)
                    })
                found = true
                break;
            }
            if (!found) {
                err = new Error("action '" + req.name + "' is not a known action for thing '" +
                    req.thingID + "'")
            }
        }
    }

    resp = req.createResponse(output, err)
    if (!err) {
        resp.status = status
    }
    if (err) {
        log.error(err)
    }
    return resp
}
