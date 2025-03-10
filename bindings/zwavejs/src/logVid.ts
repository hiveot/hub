import fs from "node:fs";
import type { TranslatedValueID, ValueMetadataNumeric, ValueMetadataString, ZWaveNode } from "zwave-js";
import type { ConfigurationMetadata, ValueMetadataBuffer } from "@zwave-js/core";
import { CommandClasses, ConfigValueFormat } from "@zwave-js/core";

import type { VidAffordance } from "./getVidAffordance.ts";
import {getVidValue} from "./ZWAPI.ts";

// Log the given vid to a CSV file.
// If vid is undefined, then write the header, otherwise the vid data.
// Intended for gathering info on zwave devices.
export default function logVid(logFd: number | undefined, node?: ZWaveNode, vid?: TranslatedValueID,
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
    const vm = node.getValueMetadata(vid)
    const dt = new Date()
    let allowManualEntry = ""
    let ccSpecific = ""
    if (vm.ccSpecific) {
        ccSpecific = JSON.stringify(vm.ccSpecific)
    }
    const defaultValue = vm.default?.toString() || ""
    let description = vm.description || ""
    let min = ""
    let max = ""
    const other = vm.valueChangeOptions?.toString() || "";
    const prop = vid.property?.toString() || ""
    const propName = vid.propertyName || ""
    const propKey = vid.propertyKey?.toString() || ""
    const propKeyName = vid.propertyKeyName || ""
    let states = ""
    const time = dt.toString()
    let dataType = vm.type
    let unit = ""
    const vidValue = getVidValue(node, vid)
    if (vid.commandClass == CommandClasses.Configuration) {
        const vmc = vm as ConfigurationMetadata
        if (vmc) {
            if (vmc.format) {
                // 0 = signed int
                // 1 = unsigned int
                // 2 = enum
                // 3 = bitField
                const formatStr = ConfigValueFormat[vmc.format].toString()
                dataType += " (" + formatStr + ")"
            }
            if (vmc.description) {
                // not clear when info is available and what it contains
                description += "; description: " + vmc.description;
            }
            if (vmc.allowManualEntry != undefined) {
                allowManualEntry = String(vmc.allowManualEntry)
            }
        }
    }

    switch (vm.type) {
        case "duration":
        case "number":
        case "color": {
            const vidNum = vm as ValueMetadataNumeric
            min = vidNum.min?.toString() || ""
            max = vidNum.max?.toString() || ""
            unit = vidNum.unit || ""
            if (vidNum.states) {
                states = JSON.stringify(vidNum.states)
            }
        } break;
        case "buffer": {
            const vidBuf = vm as ValueMetadataBuffer;
            min = vidBuf.minLength?.toString() || "";
            max = vidBuf.maxLength?.toString() || "";
        } break;
        case "string": {
            const vidStr = vm as ValueMetadataString;
            min = vidStr.minLength?.toString() || "";
            max = vidStr.maxLength?.toString() || "";
        } break;
    }


    const vidLine = `${time};${node.id};${vid?.commandClass};${vid?.commandClassName};${vid?.endpoint};` +
        `${prop};${propName};${propKey};${propKeyName};` +
        `${vidValue};${vm.label};${vm.type};${vm.readable};${vm.writeable};${defaultValue};${description};` +
        `${unit};${min};${max};${states};${allowManualEntry};${ccSpecific};${other};` +
        `${propID};${va?.vidType};${dataType};${va?.atType}\n`
    fs.appendFileSync(logFd, vidLine)
}
