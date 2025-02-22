// ZWaveJSBinding.ts holds the entry point to the zwave binding along with its configuration
import type {ZWaveNode} from "zwave-js";
import {getNodeTD} from "./getNodeTD";
import {NodeValues} from "./NodeValues";
import {ZWAPI} from "./ZWAPI.js";
import {parseController} from "./parseController";
import {logVid} from "./logVid";
import {getPropName} from "./getPropName";
import * as vocab from "@hivelib/api/vocab/vocab.js";
import fs from "fs";
import {BindingConfig} from "./BindingConfig";
import {handleRequest} from "@zwavejs/handleRequest";
import {ValueID} from "@zwave-js/core";
import {getVidAffordance} from "@zwavejs/getVidAffordance";
import {IAgentConnection} from "@hivelib/messaging/IAgentConnection";
import {RequestMessage, ResponseMessage} from "@hivelib/messaging/Messages";
import {getlogger} from "@zwavejs/getLogger";

const log = getlogger()



// ZWaveBinding maps ZWave nodes to Thing TDs and events, and handles actions to control node inputs.
//
// NOTE: zwavejs v12 train wreck:
// https://community.home-assistant.io/t/z-wave-network-lost-i-believe-after-update-to-z-wave-js-ui-v2-1-1-serial-port-error/625899
// in short, soft-reset attempt might reset the USB port and move it
// over from /dev/ttyACM0->/dev/ttyACM1 and vice versa.
// Proposed solution is to use /dev/serial/by-id, however that doesn't seem to fix it.
// for now stick with version v11.3
//
export class ZwaveJSBinding {
    // id: string = "zwave";
    hc: IAgentConnection;
    zwapi: ZWAPI;
    // the last received values for each node by deviceID
    lastValues = new Map<string, NodeValues>(); // nodeId: ValueMap

    vidCsvFD: number | undefined
    config: BindingConfig


    // @param hc: Hub client to publish and subscribe
    // @param config: binding configuration
    constructor(hc: IAgentConnection, config: BindingConfig) {
        this.hc = hc;
        this.config = config
        log.settings.minLevel = 3 // info
        // zwapi handles the zwavejs specific details
        this.zwapi = new ZWAPI(
            this.handleNodeUpdate.bind(this),
            this.handleValueUpdate.bind(this),
            this.handleNodeStateUpdate.bind(this),
            this.handleDriverError.bind(this),
        );
    }


    // Driver failed, possibly due to removal of USB stick. Restart.
    handleDriverError(e: Error): void {
        log.error("driver error",e);
    }

    // Handle update of one of the node state flags
    // This emits a corresponding event
    handleNodeStateUpdate(node: ZWaveNode, newState: string) {
        let thingID = this.zwapi.getDeviceID(node.id)

        // NOTE: the names of these values MUST match those in the TD property enum. See parseNode.
        switch (newState) {
            case "alive":
            case "dead":
            case "awake":
            case "sleeping": {
                this.hc.pubProperty(thingID, vocab.PropDeviceStatus, newState)
            }
            break;
            // FIXME: interview state as property and response
            case "interview completed":
            case "interview failed":
            case "interview started": {
                this.hc.pubEvent(thingID, "interview", newState)
            }
                break;
        }
    }

    // Handle discovery or update of a node.
    // This publishes the TD and its property values
    handleNodeUpdate(node: ZWaveNode) {
        log.info("handleNodeUpdate:node:", node.id);

        // FIXME: only update the node if it has changed
        let thingTD = getNodeTD(this.zwapi, node, this.vidCsvFD, this.config.maxNrScenes);

        if (node.isControllerNode) {
            parseController(thingTD, this.zwapi.driver.controller)
        }
        // republish the TD
        this.hc.pubTD(thingTD)

        // publish the thing property (attr, config) values
        let newValues = new NodeValues(node);
        let lastNodeValues = this.lastValues.get(thingTD.id)
        let diffValues = newValues
        if (lastNodeValues) {
            // diffValues = lastNodeValues.diffValues(newValues)
            diffValues = newValues.diffValues(lastNodeValues)
        }
        this.hc.pubMultipleProperties(thingTD.id, diffValues.values)
        this.lastValues.set(thingTD.id, newValues);

    }

    // Handle update of a node's value.
    // This publishes an event or property update if the value changed or 'publishOnlyChanges' is false
    // @param node: The node whose values have updated
    // @param vid: zwave value id
    // @param newValue: the updated value converted to a string
    handleValueUpdate(node: ZWaveNode, vid: ValueID, newValue: unknown) {
        let deviceID = this.zwapi.getDeviceID(node.id)
        let propID = getPropName(vid)
        let valueMap = this.lastValues.get(deviceID);
        // update the map of recent values
        let lastValue = valueMap?.values[propID]
        let va = getVidAffordance(node, vid, this.config.maxNrScenes)
        try {
            if (valueMap && (lastValue !== newValue || !this.config.publishOnlyChanges)) {
                // TODO: round the value using a precision
                // TODO: republish after some time even when unchanged
                // Determine if value changed enough to publish
                if (newValue != undefined) {
                    valueMap.values[propID] = newValue
                    if (va?.vidType === "property" ) {
                        this.hc.pubProperty(deviceID, propID, newValue)
                    } else {
                        log.info("handleValueUpdate: publish event for deviceID=" + deviceID + ", propID=" + propID + "")
                        this.hc.pubEvent(deviceID, propID, newValue)
                    }
                }
            } else {
                // for debugging
                log.debug("handleValueUpdate: unchanged value deviceID=" + deviceID + ", propID=" + propID + " (ignored)")

            }
        } catch (e) {
            log.error("handleValueUpdate: caught exception", e)
        }
    }

            // periodically publish the properties that have updated
    // publishPropertyUpdates() {
    //     for (let [deviceID, valueMap] of this.lastValues) {
    //         let node = this.zwapi.getNodeByDeviceID(deviceID)
    //         if (node) {
    //             let publishedValues = this.publishedValues.get(deviceID)
    //             let diffValues = publishedValues ? valueMap.diffValues(publishedValues) : valueMap;
    //             this.hapi.pubProperties(deviceID, diffValues)
    //         } else {
    //             // node no longer exist. Remove it.
    //             this.lastValues.delete(deviceID)
    //         }
    //     }
    // }


    // Starts and run the binding. 
    // This starts the zwave API/driver and invokes connect.
    // If an error is observed, invoke connect again.
    async start() {
        log.info("ZWaveJS binding start");

        // optional logging of discovered VID
        if (this.config.vidCsvFile) {
            this.vidCsvFD = fs.openSync(this.config.vidCsvFile, "w+", 0o640)
            logVid(this.vidCsvFD)
        }
        this.hc.setRequestHandler( (msg:RequestMessage):ResponseMessage => {
            let resp = handleRequest(msg,this.zwapi, this.hc)
            return resp
        })

        await this.zwapi.connectLoop(this.config);
    }

    // Stop the binding and disconnect from the ZWave controller
    async stop() {
        log.info("Shutting Down...");
        await this.zwapi.disconnect();
        if (this.vidCsvFD) {
            fs.close(this.vidCsvFD)
        }
        process.exit(0);
    }
}
