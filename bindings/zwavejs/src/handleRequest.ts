// ZWaveJSBinding.ts holds the entry point to the ZWave binding along with its configuration
import {InterviewStage,  ZWaveNode} from "zwave-js";
import type IAgentConnection from "../hivelib/messaging/IAgentConnection.ts";
import {OpWriteProperty} from "../hivelib/api/vocab/vocab.js";
import {
    NotificationMessage,
    RequestMessage,
    ResponseMessage,
    StatusCompleted,
    StatusRunning
} from "../hivelib/messaging/Messages.ts";

import ZWAPI, {getVidValue} from "./ZWAPI.ts";
import {handleWriteProperty} from "./handleWriteProperty.ts";
import setValue from "./setValue.ts";
import getLogger from "./getLogger.ts";
import {getVidFromActionName} from "./getAffordanceFromVid.ts";

const log = getLogger()



// handle request  actions as defined in the TD
//
// This sends responses asynchronously and just returns nil
export default function  handleRequest(
    req: RequestMessage, zwapi: ZWAPI, hc: IAgentConnection):ResponseMessage|null {

    let handled = handleControllerRequest(req, zwapi, hc)
    if (handled) {
        return null
    }

    const node = zwapi.getNodeByDeviceID(req.thingID)
    if (node == undefined) {
        const errMsg = new Error("handleActionRequest: node for thingID" + req.thingID + "does not exist")
        log.error(errMsg)
        return req.createResponse(null, errMsg)
    }

    if (req.operation == OpWriteProperty) {
        return handleWriteProperty(req, node, zwapi, hc)
    }

    handled = handleNodeAction(req, node, zwapi, hc)
    if (handled) {
        return null
    }

    // last, handle vid actions
    handleVidAction(req, node, hc)
    return null
}


// handle controller requests as defined in the TD.
// This returns true if the request was handled.
// This sends responses asynchronously
function  handleControllerRequest(
    req: RequestMessage, zwapi: ZWAPI, hc: IAgentConnection):boolean {

    let resp: ResponseMessage|null = null
    let handled: boolean = true

    // controller specific commands (see parseController)
    const actionLower = req.name.toLowerCase()
    switch (actionLower) {
        case "beginexclusion":
            zwapi.driver.controller.beginExclusion().then()
            resp = req.createResponse(null)

            break;
        case "begininclusion":
            zwapi.driver.controller.beginInclusion().then()
            resp = req.createResponse(null)

            break;
        case "beginrebuildingroutes":
            zwapi.driver.controller.beginRebuildingRoutes()
            resp = req.createResponse(null)
            break;

        case "removefailednode":
            let targetNode = zwapi.getNodeByDeviceID(req.thingID)
            if (targetNode) {
                zwapi.driver.controller.removeFailedNode(targetNode.id).then();
            }
            resp = req.createResponse(null)
            break;

        case "stopexclusion":
            zwapi.driver.controller.stopExclusion().then()
            resp = req.createResponse(null)
            break;

        case "stopinclusion":
            zwapi.driver.controller.stopInclusion().then()
            resp = req.createResponse(null)
            break;

        case "stoprebuildingroutes":
            zwapi.driver.controller.stopRebuildingRoutes()
            resp = req.createResponse(null)
            break;

        default:
            handled = false
    }
    if (resp) {
        hc.sendResponse(resp)
    }
    return handled
}


// handle node (Thing) actions as defined in the TD.
// This returns true if the request was handled.
// This sends responses asynchronously
function  handleNodeAction(
    req: RequestMessage, node:ZWaveNode, zwapi: ZWAPI, hc: IAgentConnection):boolean {

    let resp: ResponseMessage | null = null
    let handled: boolean = true
    let notif:NotificationMessage|undefined = undefined

    // controller specific commands (see parseController)
    const actionLower = req.name.toLowerCase()
    try {
        switch (actionLower) {
            case "checklifelinehealth":
                // 3 runs; return rating
                node.checkLifelineHealth(3)
                    .then((ev) => {
                        let resp = req.createResponse(ev.rating)
                        hc.sendResponse(resp)
                    })
                // request is running, no status neeeded
                notif = req.createNotification()
                hc.sendNotification(notif)
                break;

            case "getnodeneighbors":
                zwapi.driver.controller.getNodeNeighbors(node.id)
                    .then((ids) => {
                        resp = req.createResponse(ids)
                        hc.sendResponse(resp)
                    });
                break;

            case "ping":
                // ping a node. The 'completed' response is sent async
                const startTime = performance.now()
                node.ping()
                    .then((_success: boolean) => {
                        const endTime = performance.now()
                        const msec = Math.round(endTime - startTime)
                        let resp = req.createResponse(msec)
                        log.info("ping '" + req.thingID + "': " + msec + " msec")
                        hc.sendResponse(resp)
                    })
                break;

            case "rebuildnoderoutes":
                zwapi.driver.controller.rebuildNodeRoutes(node.id)
                    .then((success) => {
                        let err: Error | undefined
                        if (!success) {
                            err = Error("failed rebuilding node routes")
                        }
                        resp = req.createResponse(success, err)
                        hc.sendResponse(resp)
                    });
                break;

            case "refreshinfo":
                // doc warning: do not call refreshInfo when node interview is not yet complete
                if (node.interviewStage == InterviewStage.Complete) {
                    node.refreshInfo({waitForWakeup: true})
                        .then((result) => {
                            log.info("refreshinfo. StartedResult:", result)
                            let resp = req.createResponse(null)
                            hc.sendResponse(resp) // async
                        })
                    notif = req.createNotification()
                    hc.sendNotification(notif)
                } else {
                    // a previous request was still running.
                    let err = new Error("refreshinfo is already running")
                    let resp = req.createResponse(null, err)
                    hc.sendResponse(resp) // async
                }
                break;

            case "refreshvalues":
                // this can take 10-20 seconds
                node.refreshValues().then((_res) => {
                    let resp = req.createResponse(null)
                    hc.sendResponse(resp) // async
                    log.info("refreshvalues completed")
                }).catch(err => {
                    log.info("refreshvalues failed: ", err)
                    let resp = req.createResponse(null, err)
                    hc.sendResponse(resp) // async
                })
                // request is running, no status neeeded
                notif = req.createNotification()
                hc.sendNotification(notif)
            break;

            default:
                handled = false
        }
    } catch (err:any) {
        log.warn(req.operation + "failed: ", err)
        let resp = req.createResponse(null, err)
        hc.sendResponse(resp)
    }
    return handled
}


// handle Vid requests.
//
// This sends responses asynchronously
function  handleVidAction(
    req: RequestMessage, node:ZWaveNode,  hc: IAgentConnection) {

    // VID based configuration and actions
    //  currently propertyIDs are also accepted.
    // FIXME: only allow defined actions
    // FIXME: convert actionValue to expected type
    const propVid = getVidFromActionName(req.name)
    if (propVid) {
        // this is a known property
        setValue(node, propVid, req.input)
            .then(progress => {
                if (progress === StatusCompleted) {
                    const newValue = getVidValue(node, propVid)
                    let resp = req.createResponse(newValue)
                    // FIXME: this should return an ActionStatus value!
                    hc.sendResponse(resp)
                    // no longer needed
                    // zwapi.onValueUpdate(node, propVid, newValue)
                } else if (progress === StatusRunning) {
                    // FIXME: add notification support
                    // const notif = req.createNotification()
                    // hc.sendNotification(notif)
                }
            })
            .catch(err => {
                // send a failed response
                let resp = req.createResponse(null, err)
                hc.sendResponse(resp)
                log.error(err)
            })
    } else {
        let err = new Error("action '" + req.name + "' is not a known action for thing '" +
            req.thingID + "'")
        let resp = req.createResponse(null, err)
        log.error(err)
        hc.sendResponse(resp)
    }
}
