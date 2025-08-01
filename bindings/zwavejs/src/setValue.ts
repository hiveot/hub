// ZWaveJSBinding.ts holds the entry point to the zwave binding along with its configuration
import {SetValueStatus,  type ValueMetadataNumeric, ZWaveNode} from "zwave-js";
import {StatusCompleted, StatusFailed, StatusRunning} from "../hivelib/messaging/Messages.ts";

import {getEnumFromMemberName} from "./ZWAPI.ts";
import getLogger from "./getLogger.ts";
import type {ValueID} from "@zwave-js/core";

const log = getLogger()


/**
    Set a new Vid value.

    @param node node whose value to set
    @param vid valueID parameter to set
    @param value native value to set, if any
    @returns a delivery status for returning to the hub
 @throw
 */
export default async function setValue(node: ZWaveNode, vid: ValueID, value: any): Promise<string> {
    return new Promise<string>( (resolve, reject) => {
        let dataToSet: unknown
        let progress = StatusFailed
        let err:  string|undefined

        try {
            const vidMeta = node.getValueMetadata(vid)
            dataToSet = value
            if (vidMeta.type == "number") {
                const vmn = vidMeta as ValueMetadataNumeric;
                if (vmn.states) {
                    dataToSet = getEnumFromMemberName(vmn.states, value)
                }
            }

            // setVidValue(node, vid, dataToSet)
            node.setValue(vid, dataToSet, {onProgress:(_prog)=>{
                // 0: queued
                // 1: active (currently being handled)
                // 2: completed
                // 3: failed
                // log.info("setValue progress: ", prog.state.toString())
            }})
            .then(res => {
                // status: 0 - no device support
                // 1: working - command accepted and working on it
                // 2: fail - command rejected
                // 3: endpoint not found
                // 4: not implemented
                // 5: invalid value
                // 0xfe: unsupervised, success but unknown if executed
                // 0xff: success
                switch (res.status) {
                    case SetValueStatus.Working:
                        resolve(StatusRunning)
                        break;
                    // TODO progress updates
                    case SetValueStatus.Success:
                    case SetValueStatus.SuccessUnsupervised:
                        resolve(StatusCompleted)
                        break;
                    case SetValueStatus.EndpointNotFound:
                    case SetValueStatus.NotImplemented:
                        log.error(err)
                        reject(res.message)
                        break;
                    case SetValueStatus.InvalidValue:
                        log.error(err)
                        reject(res.message)
                        break
                    default:
                        // when is this invoked
                        resolve(StatusRunning)
                }
                resolve(progress)
            })
                .catch(reason=> {
                    log.error(`Failed configuring value. Reason: ${reason}`)
                    reject(reason)
                })
        } catch (reason) {
            //
            log.error(`Failed setting value. Reason: ${reason}`)
            reject(reason)
        }
    })
}