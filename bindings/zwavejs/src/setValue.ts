// ZWaveJSBinding.ts holds the entry point to the zwave binding along with its configuration
import {SetValueStatus, TranslatedValueID, ValueMetadataNumeric, ZWaveNode} from "zwave-js";
import {getEnumFromMemberName, getVidValue,  ZWAPI} from "@zwavejs/ZWAPI";
import {StatusCompleted, StatusFailed, StatusRunning} from "@hivelib/messaging/Messages";
import {getlogger} from "@zwavejs/getLogger";

const log = getlogger()



// Set a new Vid value.
// @param node: node whose value to set
// @param vid: valueID parameter to set
// @param value: native value to set, if any
// this returns a delivery status for returning to the hub
export async function setValue(node: ZWaveNode, vid: TranslatedValueID, value: any): Promise<string> {
    return new Promise<string>( (resolve, reject) => {
        let dataToSet: unknown
        let progress = StatusFailed
        let err:  Error|undefined

        try {
            let vidMeta = node.getValueMetadata(vid)
            dataToSet = value
            if (vidMeta.type == "number") {
                let vmn = vidMeta as ValueMetadataNumeric;
                if (vmn.states) {
                    dataToSet = getEnumFromMemberName(vmn.states, value)
                }
            }

            // setVidValue(node, vid, dataToSet)
            node.setValue(vid, dataToSet, {onProgress:(prog)=>{
                // 0: queued
                // 1: active (currently behing handled)
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
                            progress = StatusRunning
                            break;
                        // TODO progress updates
                        case SetValueStatus.Success:
                        case SetValueStatus.SuccessUnsupervised:
                            progress = StatusCompleted
                            break;
                        case SetValueStatus.EndpointNotFound:
                        case SetValueStatus.NotImplemented:
                            progress = StatusFailed
                            err = res.message
                            break;
                        case SetValueStatus.InvalidValue:
                            progress = StatusFailed
                            err = res.message
                            break
                        default:
                            progress = StatusRunning
                    }
                    if (err) {
                        log.error(err)
                        reject(err)
                    }
                    resolve(progress)
                })
        } catch (reason) {
            log.error(`Failed setting value. Reason: ${reason}`)
            reject(reason)
        }
    })
}