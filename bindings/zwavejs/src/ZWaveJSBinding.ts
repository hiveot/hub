// ZWaveJSBinding.ts holds the entry point to the zwave binding along with its configuration
import type {ZWaveNode} from "zwave-js";
import createNodeTD from "./createNodeTD.ts";
import NodeValues from "./NodeValues.ts";
import ZWAPI from "./ZWAPI.ts";
import createControllerTD from "./createControllerTD.ts";
import logVid from "./logVid.ts";
import * as vocab from "../hivelib/api/vocab/vocab.js";
import fs from "node:fs";
import BindingConfig from "./BindingConfig.ts";
import handleRequest from "./handleRequest.ts";
import {type ValueID} from "@zwave-js/core";
import getAffordanceFromVid from "./getAffordanceFromVid.ts";
import type IAgentConnection from "../hivelib/messaging/IAgentConnection.ts";
import {RequestMessage, ResponseMessage} from "../hivelib/messaging/Messages.ts";
import getLogger from "./getLogger.ts";
import process from "node:process";
import TD from "../hivelib/wot/TD.ts";
const log = getLogger()



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
        const thingID = this.zwapi.getDeviceID(node.id)

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

    // Handle discovery or update of a node attributes.
    // This publishes the TD and its property values
    handleNodeUpdate(node: ZWaveNode) {
        log.info("handleNodeUpdate:node:", node.id);

        // FIXME: only update the node if it has changed
        let tdi: TD

        if (node.isControllerNode) {
            tdi = createControllerTD(this.zwapi,node,this.vidCsvFD)
        } else {
            tdi = createNodeTD(this.zwapi, node, this.vidCsvFD, this.config.maxNrScenes);
        }
        // republish the TD
        this.hc.pubTD(tdi)

        // publish the thing property (attr, config) values
        const newValues = new NodeValues(node,this.zwapi.driver);
        const lastNodeValues = this.lastValues.get(tdi.id)
        let diffValues = newValues
        if (lastNodeValues) {
            // diffValues = lastNodeValues.diffValues(newValues)
            diffValues = newValues.diffValues(lastNodeValues)
        }
        this.hc.pubMultipleProperties(tdi.id, diffValues.values)
        this.lastValues.set(tdi.id, newValues);
        console.info("handleNodeUpdate completed: "+node.id)

    }

    // Handle update of a node's value.
    // This publishes an event or property update if the value changed or 'publishOnlyChanges' is false
    // @param node: The node whose values have updated
    // @param vid: zwave value id
    // @param newValue: the updated value converted to a string
    handleValueUpdate(node: ZWaveNode, vid: ValueID, newValue: unknown) {
        const deviceID = this.zwapi.getDeviceID(node.id)
        const valueMap = this.lastValues.get(deviceID);
        // update the map of recent values
        const va = getAffordanceFromVid(node, vid, this.config.maxNrScenes)
        if (!va) {
            // this is not a VID of interest so ignore it
            log.info("handleValueUpdate: unused VID ignored: CC=" + vid.commandClass,
                " vidProperty=", vid.property, "vidEndpoint=", vid.endpoint)
            return
        }
        const lastValue = valueMap?.values[va.name]
        if (valueMap && (lastValue !== newValue || !this.config.publishOnlyChanges)) {
            // TODO: round the value using a precision
            // TODO: republish after some time even when unchanged
            // Determine if value changed enough to publish
            if (newValue != undefined) {
                valueMap.values[va.name] = newValue
                if (va?.affType === "property") {
                    this.hc.pubProperty(deviceID, va.name, newValue)
                } else if (va?.affType === "action") {
                    // action output state has a matching property
                    this.hc.pubProperty(deviceID, va.name, newValue)
                } else {
                    // Anything else is an event
                    log.info("handleValueUpdate: publish event for deviceID=" + deviceID + ", propName=" + va.name + "")
                    this.hc.pubEvent(deviceID, va.name, newValue)
                }
            }
        } else {
            // for debugging
            log.debug("handleValueUpdate: unchanged value deviceID=" + deviceID + ", propName=" + va.name + " (ignored)")

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
        this.hc.setRequestHandler( (msg:RequestMessage):ResponseMessage|null => {
            const resp = handleRequest(msg,this.zwapi, this.hc)
            return resp
        })
        // the binding does not expect async responses
        // the binding does not expect any notifications

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
