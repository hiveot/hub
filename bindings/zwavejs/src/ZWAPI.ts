import {
    Driver,
    Endpoint,
    InclusionResult,
    NodeStatus,
    PartialZWaveOptions,
    RebuildRoutesStatus,
    RemoveNodeReason,
    TranslatedValueID,
    ValueMetadataNumeric,
    ZWaveNode,
    ZWaveNodeMetadataUpdatedArgs,
    ZWaveNodeValueAddedArgs,
    ZWaveNodeValueNotificationArgs,
    ZWaveNodeValueRemovedArgs,
    ZWaveNodeValueUpdatedArgs,
} from "zwave-js";
import fs, { opendir } from "fs";
import { Logger } from "tslog";
import { CommandClasses } from '@zwave-js/core';
import path from "path";
import { buffer } from "stream/consumers";

const tslog = new Logger({ name: "ZWAPI" })

// Default keys for testing, if none are configured. Please set the keys in config/zwavejs.yaml
const DefaultS2Authenticated = "00112233445566778899AABBCCDDEEFF"
const DefaultS2Unauthenticated = "112233445566778899AABBCCDDEEFF00"
const DefaultS2AccessControl = "2233445566778899AABBCCDDEEFF0011"
const DefaultS0Legacy = "33445566778899AABBCCDDEEFF001122"

// Configuration of the ZWaveJS driver
export interface IZWaveConfig {
    // These keys are generated with "< /dev/urandom tr -dc A-F0-9 | head -c32 ;echo"
    // or use MD5(password text)
    S2_Unauthenticated: string | undefined
    S2_Authenticated: string | undefined
    S2_AccessControl: string | undefined
    S0_Legacy: string | undefined
    // disable soft reset on startup. Use if driver fails to connect to the controller
    zwDisableSoftReset: boolean | undefined,
    // controller port: default auto, /dev/serial/by-id/usb...
    zwPort: string | undefined,
    zwLogFile: string | undefined       // driver logfile if any
    zwLogLevel: "error" | "warn" | "info" | "verbose" | "debug" | "",      // driver log level, "" no logging
    //
    cacheDir: string | undefined          // alternate storage directory
}


// ZWAPI is a wrapper around zwave-js for use by the HiveOT binding.
// Its primary purpose is to hide the ZWave specific logic from the binding; offer a simple API to
// obtain the data for publishing the node TDs and events; and accept actions for devices.
// To do so it transforms ZWave vocabulary to HiveOT vocabulary.
export class ZWAPI {
    // driver initializes on connect
    driver!: Driver;

    // callback to notify of fatal error, such as loss of controller connection
    onFatalError: (e: Error) => void;

    // callback to notify of a change in node VID or metadata
    onNodeUpdate: (node: ZWaveNode) => void;

    // callback to notify of a change in node state
    onStateUpdate: (node: ZWaveNode, newState: string) => void;

    // callback to notify of a change in VID value
    onValueUpdate: (node: ZWaveNode, v: TranslatedValueID, newValue: unknown) => void;

    // discovered nodes
    nodes: Map<string, ZWaveNode>;

    // doReconnect requests the background connection loop to re-initiate a connection with the controller
    doReconnect: boolean = false

    constructor(
        // handler for node VID or Metadata updates
        onNodeUpdate: (node: ZWaveNode) => void,
        // handler for node property value updates
        onValueUpdate: (node: ZWaveNode, v: TranslatedValueID, newValue: unknown) => void,
        // handler for node state updates
        onStateUpdate: (node: ZWaveNode, newState: string) => void,
        // handler for driver fatal errors
        onFatalError: (e: Error) => void) {

        this.onStateUpdate = onStateUpdate;
        this.onNodeUpdate = onNodeUpdate;
        this.onValueUpdate = onValueUpdate;
        this.onFatalError = onFatalError;
        this.nodes = new Map<string, ZWaveNode>();
    }

    // Add a known node and subscribe to its ready event before doing anything else
    addNode(node: ZWaveNode) {
        tslog.info(`Node ${node.id} - waiting for it to be ready`)
        node.on("ready", (node) => {
            tslog.info("--- Node", node.id, "is ready. Setting up the node.");
            this.setupNode(node);
        });
    }

    // connect initializes the zwave-js driver and connect it to the ZWave controller.
    //
    // @param zwConfig driver configuration
    // @param onConnectError when failing to connect to the controller device
    //  call disconnect to cancel.
    async connect(zwConfig: IZWaveConfig) {
        // autoconfig the zwave port if none given

        let zwPort = zwConfig.zwPort
        if (!zwPort) {
            tslog.info("serial port not set... searching")
            zwPort = findSerialPort()
            // zwPort = "/dev/ttyACM0"
        }
        tslog.info("connecting", "port", zwPort)

        // These keys should be generated with "< /dev/urandom tr -dc A-F0-9 | head -c32 ;echo"
        let S2_Authenticated = zwConfig.S2_Authenticated || DefaultS2Authenticated
        let S2_Unauthenticated = zwConfig.S2_Unauthenticated || DefaultS2Unauthenticated
        let S2_AccessControl = zwConfig.S2_AccessControl || DefaultS2AccessControl
        let S0_Legacy = zwConfig.S0_Legacy || DefaultS0Legacy

        let options: PartialZWaveOptions = {
            securityKeys: {
                // These keys should be generated with "< /dev/urandom tr -dc A-F0-9 | head -c32 ;echo"
                S2_Unauthenticated: Buffer.from(S2_Unauthenticated, "hex"),
                S2_Authenticated: Buffer.from(S2_Authenticated, "hex"),
                S2_AccessControl: Buffer.from(S2_AccessControl, "hex"),
                S0_Legacy: Buffer.from(S0_Legacy, "hex"),
            },
            // wait for the device verification before sending value updated event.
            //  instead some kind of 'pending' status should be tracked.
            emitValueUpdateAfterSetValue: false,
            //
            logConfig: {
                enabled: (zwConfig.zwLogLevel != ""),
                level: zwConfig.zwLogLevel,
                logToFile: !!(zwConfig.zwLogFile),
                filename: zwConfig.zwLogFile,
            },
            storage: {
                // allow for a different cache directory
                cacheDir: zwConfig.cacheDir,
            },
            // enableSoftReset: !zwConfig.zwDisableSoftReset,
            features: {
                // workaround for sticks that don't handle this and refuse connection after reset
                softReset: !zwConfig.zwDisableSoftReset,
            }

        };
        tslog.info("ZWaveJS config option soft_reset on startup is " + (options.features?.softReset ? "enabled" : "disabled"))


        // retry starting the driver until disconnect is called
        // Start the driver. To await this method, put this line into an async method
        this.driver = new Driver(zwPort, options);

        // notify of driver errors
        this.driver.on("error", (e) => {
            if (this.onFatalError) {
                this.onFatalError(e)
            }
            // reconnect
            this.doReconnect = true
        });

        // Listen for the driver ready event before doing anything with the driver
        this.driver.once("driver ready", () => {
            this.handleDriverReady()
        });
        // wrong serial port is not detected until after the return
        // this throws an error if connection failed. Up to the caller to retry
        // onError is not invoked until after a successful connect
        return this.driver.start();
    }

    // connectLoop auto-reconnects to the controller
    async connectLoop(zwConfig: IZWaveConfig) {
        this.doReconnect = true

        // every 3 seconds, check if a reconnect is needed
        setInterval(() => {
            if (this.doReconnect) {
                this.doReconnect = false
                this.connect(zwConfig)
                    .then(() => {
                        tslog.info("connectLoop: connect successful")
                    })
                    .catch((e) => {
                        tslog.error("connectLoop: no connection with controller");
                        //retry
                        this.doReconnect = true
                    })
            }
        }, 3000);
    }


    // disconnect from the ZWave controller
    async disconnect() {
        if (this.driver) {
            await this.driver.destroy();
        }
    }

    // // return the deviceID for the given Node
    // getDeviceID(node: ZWaveNode): string {
    //   return getDeviceID(this.driver, node)
    // }

    // Create the unique device ID for publishing
    // @param nodeID as provided by the node
    getDeviceID(nodeID: number): string {
        let deviceID: string = this.homeID + "." + nodeID.toString();
        return deviceID
    }

    // return the node for the given deviceID or undefined if the deviceID is not known
    // @param deviceID as provided by getDeviceID
    getNodeByDeviceID(deviceID: string): ZWaveNode | undefined {
        let parts = deviceID.split(".")
        if (parts.length != 2) {
            return undefined
        }
        let nodeID = Number(parts[1])
        let node = this.driver.controller.nodes.get(nodeID)
        return node
    }

    // return the map of discovered ZWave nodes
    // getNodes(): ReadonlyThrowingMap<number, ZWaveNode> {
    //   return this.driver.controller.nodes
    // }


    // Driver is ready.
    handleDriverReady() {

        /*
          Now the controller interview is complete. This means we know which nodes
          are included in the network, but they might not be ready yet.
          The node interview will continue in the background.
          */
        let ctl = this.driver.controller;
        // homeID is ready after the controller interview
        // this.homeID = ctl.homeId ? ctl.homeId.toString(16).toUpperCase() : "n/a";

        tslog.info("Cache Dir: ", this.driver.cacheDir);
        tslog.info("Home ID:   ", this.driver.controller.homeId?.toString(16));

        ctl.nodes.forEach((node) => {
            // Subscribe to each node to catch its ready event.
            this.addNode(node);
        });

        // controller emitted events
        this.driver.controller.on("exclusion failed", () => {
            tslog.info("exclusion has failed");
        });
        this.driver.controller.on("exclusion started", () => {
            tslog.info("exclusion has started");
        });
        this.driver.controller.on("exclusion stopped", () => {
            tslog.info("exclusion has stopped");
        });

        this.driver.controller.on("rebuild routes progress",
            (progress: ReadonlyMap<number, RebuildRoutesStatus>) => {
                tslog.info("rebuild routes progress:", progress);
            });
        this.driver.controller.on("rebuild routes done", () => {
            tslog.info("rebuild routes done");
        });

        this.driver.controller.on("inclusion failed", () => {
            tslog.info("inclusion has failed");
        });
        this.driver.controller.on("inclusion started", (secure: boolean) => {
            tslog.info("inclusion has started. secure=%v", secure);
        });
        this.driver.controller.on("inclusion stopped", () => {
            tslog.info("inclusion has stopped");
        });

        this.driver.controller.on("node added", (node: ZWaveNode, result: InclusionResult) => {
            result.lowSecurity
            tslog.info(`new node added: nodeId=${node.id} lowSecurity=${result.lowSecurity}`)
            this.setupNode(node);
        });
        this.driver.controller.on("node removed", (node: ZWaveNode, reason: RemoveNodeReason) => {
            tslog.info(`node removed: id=${node.id}, reason=${reason}`);
        });

    }


    // return the homeID of the driver
    // This returns the homeID as hex string
    get homeID(): string {
        let hid = this.driver.controller.homeId;
        let homeIDStr = hid?.toString(16).toUpperCase() || "n/a"
        return homeIDStr
    }

    // setup a new node after it is ready
    setupNode(node: ZWaveNode) {
        // first time publish node TD and value map
        this.onNodeUpdate?.(node);

        node.on("alive", (node: ZWaveNode, oldStatus: NodeStatus) => {
            tslog.info(`Node ${node.id}: is alive`);
            if (node.status != oldStatus) {
                this.onStateUpdate(node, "alive")
            }
        });
        node.on("dead", (node: ZWaveNode, oldStatus: NodeStatus) => {
            tslog.info(`Node ${node.id}: is dead`);
            if (node.status != oldStatus) {
                this.onStateUpdate(node, "dead")
            }
        });

        node.on("interview completed", (node: ZWaveNode) => {
            tslog.info(`Node ${node.id}: interview completed`);
            // event
            this.onStateUpdate(node, "interview completed")
        });
        node.on("interview failed", (node: ZWaveNode) => {
            tslog.info(`Node ${node.id}: interview failed`);
            this.onStateUpdate(node, "interview failed")
        });
        node.on("interview started", (node: ZWaveNode) => {
            tslog.info(`Node ${node.id}: interview started`);
            this.onStateUpdate(node, "interview started")
        });

        node.on("metadata updated", (node: ZWaveNode, args: ZWaveNodeMetadataUpdatedArgs) => {
            // FIXME: this is invoked even when metadata isn't updated. What to do?
            // this.onNodeUpdate(node)
            let newValue = node.getValue(args)
            this.onValueUpdate(node, args, newValue)
            tslog.info(`Node ${node.id} value metadata updated`,
                "property", args.property,
                "propkeyname", args.propertyKeyName,
                "newValue", newValue,
            );
        });

        node.on("notification", (endpoint: Endpoint, cc: CommandClasses, args) => {
            tslog.info(`Node ${endpoint.nodeId} Notification: CC=${cc}, args=${args}`)
            // TODO: what/when is this notification providing?
        });

        node.on("sleep", (node: ZWaveNode) => {
            tslog.info(`Node ${node.id}: is sleeping`);
            this.onStateUpdate(node, "sleeping")
        });

        // node.on("statistics updated", (node: ZWaveNode, args: NodeStatistics) => {
        //     // tslog.info("Node ", node.id, " stats updated: args=", args);
        // });

        node.on("value added", (node: ZWaveNode, args: ZWaveNodeValueAddedArgs) => {
            tslog.info(`Node ${node.id}, value added: propName=${args.propertyName}, value=${args.newValue}`);
            this.onNodeUpdate(node)
            this.onValueUpdate(node, args, args.newValue)
        });

        node.on("value notification", (node: ZWaveNode, vid: ZWaveNodeValueNotificationArgs) => {
            tslog.info(`Node ${node.id}, value notification: propName=${vid.propertyName}, value=${vid.value}`);
            this.onValueUpdate(node, vid, vid.value)
        });

        node.on("value removed", (node: ZWaveNode, args: ZWaveNodeValueRemovedArgs) => {
            tslog.info("Node ", node.id, " value removed for ", args.propertyName, ":", args.prevValue);
            this.onValueUpdate(node, args, undefined)
        });

        node.on("value updated", (node: ZWaveNode, args: ZWaveNodeValueUpdatedArgs) => {
            // tslog.info("Node ", node.id, " value updated: args=", args, "vidMeta=", vidMeta);
            this.onValueUpdate(node, args, args.newValue)
        });

        node.on("wake up", (node: ZWaveNode) => {
            tslog.info(`Node ${node.id}: wake up`);
            this.onStateUpdate(node, "awake")
        });
    }


    // Set the Vid value
    // This converts the given value into the right format
    // @param node: node whose value to set
    // @param vid: valueID parameter to set
    // @param params: parameters containing the value(s) to set
    setValue(node: ZWaveNode, vid: TranslatedValueID, params: string) {
        let dataToSet: unknown
        let vidMeta = node.getValueMetadata(vid)

        switch (vidMeta.type) {
            case "boolean":
                dataToSet = !(params.toLowerCase() == "false" || params == "0")
                break;
            case "number":
            case "color":
            case "duration":
                // convert enum names to values
                let numMeta = vidMeta as ValueMetadataNumeric
                if (numMeta && numMeta.states) {
                    dataToSet = getEnumFromMemberName(numMeta.states, params)
                } else {
                    dataToSet = Number(params)
                }
                if (isNaN(dataToSet as number)) {
                    dataToSet = undefined
                }
                break;
            case "string":
                dataToSet = String(params);
                break
            case "any":
                dataToSet = params;
                break;
            case "boolean[]":
            case "buffer":
            case "number[]":
            case "string[]":
                // TODO: handle arrays
                tslog.error(`setValue: Unsupported type ${vidMeta.type}`)
                break;
        }
        if (dataToSet != undefined) {
            node.setValue(vid, dataToSet, {})
                .then((accepted) => {
                    if (accepted) {
                        // success
                    } else {
                        tslog.error("setValue: failed. (why?)")
                    }
                })
                .catch((reason) => {
                    tslog.error(`Failed setting value. Reason: ${reason}`)
                })
        }
    }

}

// Determine which serial port is available
function findSerialPort(): string {
    const serialDir = "/dev/serial/by-id/"
    try {

        const dir = fs.opendirSync(serialDir);
        let first = dir.readSync()
        if (first != null) {
            return path.join(serialDir, first.name)
        }
    } catch (err) {
        console.error(err);
    }

    // force an error
    return "/dev/serialportnotfound"

}


// Revert the enum name to its number
// This is the opposite of getEnumMemberName
function getEnumFromMemberName(enumeration: Record<number, string>, name: string): number | undefined {
    for (let key in enumeration) {
        let val = enumeration[key]
        if (val.toLowerCase() == name.toLowerCase()) {
            return Number(key)
        }
    }
    // in case the enum is optional and the name is a number already
    return Number(name)
}
