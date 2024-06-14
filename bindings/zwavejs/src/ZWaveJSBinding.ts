// ZWaveJSBinding.ts holds the entry point to the zwave binding along with its configuration
import type {TranslatedValueID, ZWaveNode} from "zwave-js";
import {InterviewStage} from "zwave-js";
import {parseNode} from "./parseNode";
import {ParseValues} from "./ParseValues";
import {ZWAPI} from "./ZWAPI.js";
import {parseController} from "./parseController";
import {logVid} from "./logVid";
import {getPropKey} from "./getPropKey";
import * as vocab from "@hivelib/api/vocab/ht-vocab";
import {ActionTypeProperties} from "@hivelib/api/vocab/ht-vocab";
import fs from "fs";
import {ThingMessage} from "@hivelib/things/ThingMessage";
import {BindingConfig} from "./BindingConfig";
import * as tslog from 'tslog';
import {DeliveryProgress, DeliveryStatus, IHubClient} from "@hivelib/hubclient/IHubClient";

const log = new tslog.Logger()


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
    hc: IHubClient;
    zwapi: ZWAPI;
    // the last received values for each node by deviceID
    lastValues = new Map<string, ParseValues>(); // nodeId: ValueMap

    vidCsvFD: number | undefined
    config: BindingConfig


    // @param hapi: connectd Hub API to publish and subscribe
    // @param vidLogFile: optional csv file to write discovered VID and metadata records
    constructor(hc: IHubClient, config: BindingConfig) {
        this.hc = hc;
        this.config = config
        // zwapi handles the zwavejs specific details
        this.zwapi = new ZWAPI(
            this.handleNodeUpdate.bind(this),
            this.handleValueUpdate.bind(this),
            this.handleNodeStateUpdate.bind(this),
            this.handleDriverError.bind(this),
        );
    }


    // handle controller actions as defined in the TD
    handleActionRequest(tm: ThingMessage): DeliveryStatus {
        let stat = new DeliveryStatus()

        let actionLower = tm.key.toLowerCase()
        let targetNode: ZWaveNode | undefined
        let node = this.zwapi.getNodeByDeviceID(tm.thingID)
        if (node == undefined) {
            let errMsg = new Error("handleActionRequest: node for thingID" + tm.thingID + "does not exist")
            stat.Failed(tm,errMsg)
            return stat
        }
        if (tm.key == ActionTypeProperties) {
            return this.handleConfigRequest(tm)
        }
        log.info("action:" + tm.key)
        // controller specific commands (see parseController)
        switch (actionLower) {
            case "begininclusion":
                this.zwapi.driver.controller.beginInclusion().then()
                break;
            case "stopinclusion":
                this.zwapi.driver.controller.stopInclusion().then()
                break;
            case "beginexclusion":
                this.zwapi.driver.controller.beginExclusion().then()
                break;
            case "stopexclusion":
                this.zwapi.driver.controller.stopExclusion().then()
                break;

            case "beginrebuildingroutes":
                this.zwapi.driver.controller.beginRebuildingRoutes()
                break;
            case "stoprebuildingroutes":
                this.zwapi.driver.controller.stopRebuildingRoutes()
                break;
            case "getnodeneighbors": // param nodeID
                targetNode = this.zwapi.getNodeByDeviceID(tm.thingID)
                if (targetNode) {
                    this.zwapi.driver.controller.getNodeNeighbors(targetNode.id).then();
                }
                break;
            case "rebuildnoderoutes": // param nodeID
                targetNode = this.zwapi.getNodeByDeviceID(tm.thingID)
                if (targetNode) {
                    this.zwapi.driver.controller.rebuildNodeRoutes(targetNode.id).then();
                }
                break;
            case "removefailednode": // param nodeID
                targetNode = this.zwapi.getNodeByDeviceID(tm.thingID)
                if (targetNode) {
                    this.zwapi.driver.controller.removeFailedNode(targetNode.id).then();
                }
                break;
            // Special management actions that are accessible by writing configuration updates that are not VIDs
            // case PropTypes.Name.toLowerCase():  // FIXME: what is this. set name ???
            //     node.name = params;
            //     break;
            case "checklifelinehealth":
                node.checkLifelineHealth().then()
                break;
            case "ping":
                node.ping().then((success) => {
                    this.hc.pubEvent(tm.thingID, "ping", success ? "success" : "fail")
                })
                break;
            case "refreshinfo":
                // do not use when node interview is not yet complete
                if (node.interviewStage == InterviewStage.Complete) {
                    node.refreshInfo({waitForWakeup: true}).then()
                }
                break;
            case "refreshvalues":
                node.refreshValues().then()
                break;
            default:
                // VID based configuration and actions
                //  currently propertyIDs are also accepted.
                for (let vid of node.getDefinedValueIDs()) {
                    let propID = getPropKey(vid)
                    if (propID.toLowerCase() == actionLower) {
                        this.zwapi.setValue(node, vid, tm.data)
                        break;
                    }
                }
        }
        stat.Completed(tm)
        return stat
    }

    // handle configuration requests as defined in the TD
    handleConfigRequest(tv: ThingMessage):  DeliveryStatus {
        let stat = new DeliveryStatus()
        stat.error = "todo handle config"
        stat.status = DeliveryProgress.DeliveryFailed
        return stat
    }

    // Driver failed, possibly due to removal of USB stick. Restart.
    handleDriverError(e: Error): void {
        log.error("driver error");
    }

    // Handle update of one of the node state flags
    // This emits a corresponding event
    handleNodeStateUpdate(node: ZWaveNode, newState: string) {
        let thingID = this.zwapi.getDeviceID(node.id)

        // NOTE: the names of these events and state MUST match those in the TD event enum. See parseNode.
        switch (newState) {
            case "alive":
            case "dead":
            case "awake":
            case "sleeping": {
                this.hc.pubEvent(thingID, vocab.PropDeviceStatus, newState)
            }
                break;
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
        let thingTD = parseNode(this.zwapi, node, this.vidCsvFD, this.config.maxNrScenes);

        if (node.isControllerNode) {
            parseController(thingTD, this.zwapi.driver.controller)
        }
        // republish the TD
        this.hc.pubTD(thingTD)

        // publish the thing property (attr, config) values
        let newValues = new ParseValues(node);
        let lastNodeValues = this.lastValues.get(thingTD.id)
        let diffValues = newValues
        if (lastNodeValues) {
            diffValues = lastNodeValues.diffValues(newValues)
        }
        this.hc.pubProps(thingTD.id, diffValues.values).then()
        this.lastValues.set(thingTD.id, newValues);

    }

    // Handle update of a node's value.
    // This publishes an event if the value changed or 'publishOnlyChanges' is false
    // @param node: The node whos values have updated
    // @param vid: zwave value id
    // @param newValue: the updated value converted to a string
    handleValueUpdate(node: ZWaveNode, vid: TranslatedValueID, newValue: unknown) {
        let deviceID = this.zwapi.getDeviceID(node.id)
        let propID = getPropKey(vid)
        let valueMap = this.lastValues.get(deviceID);
        // update the map of recent values
        let lastValue = valueMap?.values[propID]
        if (valueMap && (lastValue !== newValue || !this.config.publishOnlyChanges)) {
            // TODO: convert the value to text using a precision
            // Determine if value changed enough to publish
            if (newValue != undefined) {
                valueMap.values[propID] = newValue.toString()
                //
                let serValue = JSON.stringify(newValue)
                log.info("handleValueUpdate: publish event for deviceID=" + deviceID + ", propID=" + propID + "")
                this.hc.pubEvent(deviceID, propID, serValue)
            }
        } else {
            // for debugging
            log.info("handleValueUpdate: unchanged value deviceID="+deviceID+", propID="+propID+" (ignored)" )

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
        this.hc.setActionHandler(this.handleActionRequest)

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
