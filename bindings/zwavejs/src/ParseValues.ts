import {getEnumMemberName, NodeStatus, ZWaveNode, ZWavePlusNodeType, ZWavePlusRoleType,} from "zwave-js";
import {InterviewStage, SecurityClass} from '@zwave-js/core';
import * as vocab from "@hivelib/api/vocab/ht-vocab";
import {getPropKey} from "./getPropKey";


// Value map for node values
export class ParseValues {
    values: { [key: string]: string }

    // @param node: create the map for this node
    constructor(node?: ZWaveNode) {
        this.values = {}
        if (node) {
            this.parseNodeValues(node)
        }
    }


    // Set a value if it is not undefined
    setIf(key: string, val: unknown) {
        if (val != undefined) {
            this.values[key] = val.toString()
        }
    }

    // compare the current value map with an old map and return a new value map with the differences 
    // This only returns values if they exist in the current map.
    diffValues(oldValues: ParseValues): ParseValues {
        let diffMap = new ParseValues()
        for (let key in Object(this.values)) {
            let oldVal = oldValues.values[key]
            let newVal = this.values[key]
            if (newVal !== oldVal) {
                diffMap.values[key] = newVal
            }
        }
        return diffMap
    }

    // parseNodeValues updates the value map with the latest node value
    parseNodeValues(node: ZWaveNode) {

        //--- Node read-only attributes that are common to many nodes
        this.setIf("associationCount", node.deviceConfig?.associations?.size);
        this.setIf("canSleep", node.canSleep);
        this.setIf("deviceDatabaseURL", node.deviceDatabaseUrl);
        this.setIf(vocab.PropDeviceDescription, node.deviceConfig?.description);

        if (node.deviceClass) {
            this.setIf("deviceClassBasic", node.deviceClass.basic.label);
            this.setIf("deviceClassGeneric", node.deviceClass.generic.label);
            this.setIf("deviceClassSpecific", node.deviceClass.specific.label);
            // this.setIf("supportedCCs", node.deviceClass.generic.supportedCCs);
        }
        this.setIf("endpointCount", node.getEndpointCount().toString());
        // this.setIf("dc.firmwareVersion", node.deviceConfig?.firmwareVersion);
        this.setIf(vocab.PropDeviceFirmwareVersion, node.firmwareVersion?.toString());

        if (node.getHighestSecurityClass()) {
            let classID = node.getHighestSecurityClass() as number
            let highestSecClass = `${getEnumMemberName(SecurityClass, classID)} (${classID})`
            this.setIf("highestSecurityClass", highestSecClass);
        }

        this.setIf("interviewAttempts", node.interviewAttempts);
        this.setIf("interviewStage", getEnumMemberName(InterviewStage, node.interviewStage));
        this.setIf("isListening", node.isListening);
        this.setIf("isSecure", node.isSecure);
        this.setIf("isRouting", node.isRouting);
        this.setIf("isControllerNode", node.isControllerNode)
        this.setIf("keepAwake", node.keepAwake);

        this.setIf(vocab.PropDeviceTitle, node.name)
        this.setIf(vocab.PropLocation, node.location)

        // this.setIf("label", node.deviceConfig?.label)
        this.setIf("nodeLabel", node.label)
        this.setIf("manufacturerId", node.manufacturerId);
        this.setIf(vocab.PropDeviceMake, node.deviceConfig?.manufacturer);

        this.setIf("maxDataRate", node.maxDataRate)
        if (node.nodeType) {
            this.setIf("nodeType", node.nodeType);
            this.setIf("nodeTypName", getEnumMemberName(ZWavePlusNodeType, node.nodeType));
        }
        this.setIf("paramCount", node.deviceConfig?.paramInformation?.size);
        this.setIf("productId", node.productId);
        this.setIf("productType", node.productType);
        this.setIf("protocolVersion", node.protocolVersion);
        this.setIf(vocab.PropDeviceSoftwareVersion, node.sdkVersion);
        this.setIf(vocab.PropDeviceStatus, getEnumMemberName(NodeStatus, node.status));
        this.setIf("supportedDataRates", node.supportedDataRates);

        this.setIf("userIcon", node.userIcon);
        if (node.zwavePlusNodeType) {
            this.setIf("zwavePlusNodeType", node.zwavePlusNodeType);
            this.setIf("zwavePlusNodeTypeName", getEnumMemberName(ZWavePlusNodeType, node.zwavePlusNodeType));
        }
        if (node.zwavePlusRoleType) {
            this.setIf("zwavePlusRoleType", node.zwavePlusRoleType);
            this.setIf("zwavePlusRoleTypeName", getEnumMemberName(ZWavePlusRoleType, node.zwavePlusRoleType));
        }
        this.setIf("zwavePlusVersion", node.zwavePlusVersion);

        // add value ID values
        let vids = node.getDefinedValueIDs()
        for (let vid of vids) {
            let vidValue = node.getValue(vid)
            let propID = getPropKey(vid)
            this.setIf(propID, vidValue)
        }
        // let nameVid = {
        //     commandClass:0x77, endpoint:0, property: "name"}
        // let cname = node.getValue(nameVid)
        // if (cname) {
        //     console.log("Found it!")
        // }
    }

}

