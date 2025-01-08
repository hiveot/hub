// getDeviceType returns the device type of the node in the HiveOT vocabulary
// this is based on the generic device class name. eg 'Binary Switch' and will be converted
// to the HiveOT vocabulary.
import type {ZWaveNode} from "zwave-js";

export function getDeviceType(node: ZWaveNode): string {
    let deviceClassGeneric = node.deviceClass?.generic.label;
    let deviceType: string;

    deviceType = deviceClassGeneric ? deviceClassGeneric : node.name ? node.name : "n/a";

    // TODO: map the zwave CC to the HiveOT vocabulary

    return deviceType
}
