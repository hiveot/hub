// ZWaveJSBinding.ts holds the entry point to the zwave binding along with its configuration
import {SetValueStatus, TranslatedValueID, ValueMetadataNumeric, ZWaveNode} from "zwave-js";
import * as tslog from 'tslog';
import {DeliveryProgress, DeliveryStatus, IHubClient} from "@hivelib/hubclient/IHubClient";
import {getEnumFromMemberName, getVidValue,  ZWAPI} from "@zwavejs/ZWAPI";

const log = new tslog.Logger()



// Set a new Vid value.
// @param node: node whose value to set
// @param vid: valueID parameter to set
// @param value: native value to set, if any
// this returns a delivery status for returning to the hub
export async function setValue(node: ZWaveNode, vid: TranslatedValueID, value: any): Promise<DeliveryStatus> {
    return new Promise<DeliveryStatus>( (resolve, reject) => {
        let dataToSet: unknown
        let stat = new DeliveryStatus()
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
                    resolve(stat)
                })
        } catch (reason) {
            log.error(`Failed setting value. Reason: ${reason}`)
            reject(reason)
        }
    })
}