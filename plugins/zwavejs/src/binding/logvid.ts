// Log the given VID to a CSV file.
// If vid is undefined, then write the header, otherwise the vid data
import type {TranslatedValueID, ValueMetadataNumeric, ValueMetadataString, ZWaveNode} from "zwave-js";
import type {ConfigurationMetadata, ValueMetadataBuffer} from "@zwave-js/core";
import {CommandClasses, ConfigValueFormat} from "@zwave-js/core";
import type {VidAffordance} from "./getVidAffordance";

export function logVid(logFd: number | undefined, node?: ZWaveNode, vid?: TranslatedValueID,
                       propID?: string, va?: VidAffordance) {
    if (!logFd) {
        return
    } else if (!node || !vid) {
        fs.appendFileSync(logFd,
            "time; nodeID; CC; CC Name; endpoint; property; propertyName; propertyKey; propertyKeyName; " +
            "value; label; type; readable; writable; default; description; " +
            "unit; min; max; states; allowManualEntry; ccSpecific; other; " +
            "propID; affordance; dataType; atType\n");
        return
    }
    let vm = node.getValueMetadata(vid)
    let dt = new Date()
    let allowManualEntry = ""
    let ccSpecific = ""
    if (vm.ccSpecific) {
        ccSpecific = JSON.stringify(vm.ccSpecific)
    }
    let defaultValue = vm.default?.toString() || ""
    let description = vm.description || ""
    let format = ""
    let min = ""
    let max = ""
    let other = ""
    let prop = vid.property?.toString() || ""
    let propName = vid.propertyName || ""
    let propKey = vid.propertyKey?.toString() || ""
    let propKeyName = vid.propertyKeyName || ""
    let states = ""
    let time = dt.toString()
    let dataType = vm.type
    let unit = ""
    let vidValue = node.getValue(vid)
    if (vid.commandClass == CommandClasses.Configuration) {
        let vmc = vm as ConfigurationMetadata
        if (vmc) {
            if (vmc.format) {
                // 0 = signed int
                // 1 = unsigned int
                // 2 = enum
                // 3 = bitField
                let formatStr = ConfigValueFormat[vmc.format].toString()
                dataType += " (" + formatStr + ")"
            }
            if (vmc.info) {
                // not clear when info is available and what it contains
                description += "; info: " + vmc.info;
            }
            if (vmc.allowManualEntry != undefined) {
                allowManualEntry = String(vmc.allowManualEntry)
            }
        }
    }

    switch (vm.type) {
        case "duration":
        case "number" :
        case "color" :
            let vidNum = vm as ValueMetadataNumeric
            min = vidNum.min?.toString() || ""
            max = vidNum.max?.toString() || ""
            unit = vidNum.unit || ""
            if (vidNum.states) {
                states = JSON.stringify(vidNum.states)
            }
            break;
        case "buffer":
            let vidBuf = vm as ValueMetadataBuffer;
            min = vidBuf.minLength?.toString() || "";
            max = vidBuf.maxLength?.toString() || "";
            break;
        case "string":
            let vidStr = vm as ValueMetadataString;
            min = vidStr.minLength?.toString() || "";
            max = vidStr.maxLength?.toString() || "";
            break;
    }
    other = vm.valueChangeOptions?.toString() || "";


    let vidLine = `${time};${node.id};${vid?.commandClass};${vid?.commandClassName};${vid?.endpoint};` +
        `${prop};${propName};${propKey};${propKeyName};` +
        `${vidValue};${vm.label};${vm.type};${vm.readable};${vm.writeable};${defaultValue};${description};` +
        `${unit};${min};${max};${states};${allowManualEntry};${ccSpecific};${other};` +
        `${propID};${va?.affordance};${dataType};${va?.atType}\n`
    fs.appendFileSync(logFd, vidLine)
}
