import { Buffer } from "node:buffer";

import getLogger from "./getLogger.ts";
import findSerialPort from "./serial/findserialport.ts";
import {
    Driver, FoundNode,
    InclusionResult,
    InclusionStrategy,
    NodeStatus,
    PartialZWaveOptions,
    RebuildRoutesStatus,
    RemoveNodeReason,
    ValueID, ValueMetadataNumeric,
    ZWaveNode, ZWaveNodeMetadataUpdatedArgs,
    ZWaveNodeValueAddedArgs,
    ZWaveNodeValueNotificationArgs,
    ZWaveNodeValueRemovedArgs,
    ZWaveNodeValueUpdatedArgs
} from "zwave-js";

// fix 'require is not defined' when running as a module
// This comes from zwave-js-server: server.ts
// import { createRequire } from "node:module";
// export var require = createRequire(import.meta.url);


// Default keys for testing, if none are configured. Please set the keys in config/zwavejs.yaml
const DefaultS2AccessControl = "2233445566778899AABBCCDDEEFF0011"
const DefaultS2Authenticated = "00112233445566778899AABBCCDDEEFF"
const DefaultS2Unauthenticated = "112233445566778899AABBCCDDEEFF00"
const DefaultS0Legacy = "33445566778899AABBCCDDEEFF001122"
const DefaultS2LRAccessControl = "2233445566778899AABBCCFFFEEE0011"
const DefaultS2LRAuthenticated = "00112233445566778899EEEFFFDDEEFF"

// Configuration of the ZWaveJS driver
export interface IZWaveConfig {
    // These keys are generated with "< /dev/urandom tr -dc A-F0-9 | head -c32 ;echo"
    // or use MD5(password text)
    S0_Legacy: string | undefined
    S2_Unauthenticated: string | undefined
    S2_Authenticated: string | undefined
    S2_AccessControl: string | undefined
    S2LR_AccessControl: string | undefined  // zwave long range keys
    S2LR_Authenticated: string | undefined  // zwave long range keys

    // disable soft reset on startup. Use if driver has difficulty connecting to the controller
    zwDisableSoftReset: boolean | undefined,
    // controller port: default auto, /dev/serial/by-id/usb...
    zwPort: string | undefined,
    zwLogFile: string | undefined       // driver logfile if any
    zwLogLevel: "error" | "warn" | "info" | "verbose" | "debug" | "",      // driver log level, "" no logging
    //
    cacheDir: string | undefined          // alternate storage directory
}

const log = getLogger()

// ZWAPI is a wrapper around zwave-js for use by the HiveOT binding.
// Its primary purpose is to hide the ZWave specific logic from the binding; offer a simple API to
// obtain the data for publishing the node TDs and events; and accept actions for devices.
// To do so it transforms ZWave vocabulary to HiveOT vocabulary.
export default class ZWAPI {
    // driver initializes on connect
    driver!: Driver;

    // callback to notify of fatal error, such as loss of controller connection
    onFatalError: (e: Error) => void;

    // callback to notify of a change in node VID or metadata
    onNodeUpdate: (node: ZWaveNode) => void;

    // callback to notify of a change in node state
    onStateUpdate: (node: ZWaveNode, newState: string) => void;

    // callback to notify of a change in VID value
    // note that enum values have been converted automatically to their text representation
    onValueUpdate: (node: ZWaveNode, v: ValueID, newValue: unknown) => void;

    // discovered nodes
    // nodes: Map<string, ZWaveNode>;

    // doReconnect requests the background connection loop to re-initiate a connection with the controller
    doReconnect: boolean = false

    constructor(
        // handler for node VID or Metadata updates
        onNodeUpdate: (node: ZWaveNode) => void,
        // handler for node property value updates
        onValueUpdate: (node: ZWaveNode, v: ValueID, newValue: unknown) => void,
        // handler for node state updates
        onStateUpdate: (node: ZWaveNode, newState: string) => void,
        // handler for driver fatal errors
        onFatalError: (e: Error) => void) {

        this.onStateUpdate = onStateUpdate;
        this.onNodeUpdate = onNodeUpdate;
        this.onValueUpdate = onValueUpdate;
        this.onFatalError = onFatalError;
        // this.nodes = new Map<string, ZWaveNode>();
    }

    // Add a known node and subscribe to its ready event before doing anything else
    addNode(node: ZWaveNode) {
        log.info(`Node ${node.id} - waiting for it to be ready`)

        // workaround for node interview not completing.
        // requesting a refresh seems to help
        node.refreshInfo()
            .then(res=>{
                log.info("AddNode refresh completed", res)
            })
            .catch(err=>{
                log.info("AddNode refresh failed", err)
        })

        // subscribe to the node events
        // Warning: looks like nodes don't get ready in the latest zwave-js driver
        this.setupNode(node);
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
            log.info("serial port not set... searching")
            zwPort = findSerialPort()
            // zwPort = "/dev/ttyACM0"
        }
        log.info("connecting", "port", zwPort)

        // These keys should be generated with "< /dev/urandom tr -dc A-F0-9 | head -c32 ;echo"
        const S0_Legacy = zwConfig.S0_Legacy || DefaultS0Legacy
        const S2_AccessControl = zwConfig.S2_AccessControl || DefaultS2AccessControl
        const S2_Authenticated = zwConfig.S2_Authenticated || DefaultS2Authenticated
        const S2_Unauthenticated = zwConfig.S2_Unauthenticated || DefaultS2Unauthenticated
        const S2LR_AccessControl = zwConfig.S2LR_AccessControl || DefaultS2LRAccessControl
        const S2LR_Authenticated = zwConfig.S2LR_Authenticated || DefaultS2LRAuthenticated

        const options: PartialZWaveOptions = {
            attempts: {
                controller: 3,
                nodeInterview: 10,
            },
            emitValueUpdateAfterSetValue: true,
            logConfig: {
                enabled: (zwConfig.zwLogLevel != ""),
                level: zwConfig.zwLogLevel,
                logToFile: !!(zwConfig.zwLogFile),
                filename: zwConfig.zwLogFile,
            },
            features: {
                // workaround for sticks that don't handle this and refuse connection after reset
                softReset: !zwConfig.zwDisableSoftReset,
                unresponsiveControllerRecovery: true
            },
            securityKeys: {
                // These keys should be generated with "< /dev/urandom tr -dc A-F0-9 | head -c32 ;echo"
                S0_Legacy: Buffer.from(S0_Legacy, "hex"),
                S2_AccessControl: Buffer.from(S2_AccessControl, "hex"),
                S2_Authenticated: Buffer.from(S2_Authenticated, "hex"),
                S2_Unauthenticated: Buffer.from(S2_Unauthenticated, "hex"),
            },
            securityKeysLongRange: {
                S2_AccessControl: Buffer.from(S2LR_AccessControl, "hex"),
                S2_Authenticated: Buffer.from(S2LR_Authenticated, "hex"),
            },
            storage: {
                // allow for a different cache directory
                cacheDir: zwConfig.cacheDir,
            },
            timeouts: {
              ack: 10000,   // how long to wait for an ack - 10sec for testing
                retryJammed: 2000,
                report: 2000,
                sendToSleep: 500
            }
        };
        log.info("ZWaveJS config option soft_reset on startup is " + (options.features?.softReset ? "enabled" : "disabled"))

        // retry starting the driver until disconnect is called
        // Start the driver. To await this method, put this line into an async method
        this.driver = new Driver(zwPort, options);

        // driver.configVersion causes panic "require is not defined" when using modules
        // log.info("Starting zwave-js. Version="+this.driver.configVersion)

        // notify of driver errors
        this.driver.on("error", (e:any) => {
            if (this.onFatalError) {
                this.onFatalError(e)
            }
            // reconnect
            this.doReconnect = true
        });

        // Listen for the driver ready event before doing anything with the driver
        this.driver.once("driver ready", () => {
            log.info("driver ready")
            this.handleDriverReady()
        });
        this.driver.once("all nodes ready", () => {
            log.info("all nodes ready")
            // this.handleDriverReady()
        });
        // wrong serial port is not detected until after the return
        // this throws an error if connection failed. Up to the caller to retry
        // onError is not invoked until after a successful connect
        await this.driver.start();
        log.info("driver started")
    }

    // connectLoop auto-reconnects to the controller
    async connectLoop(zwConfig: IZWaveConfig) {
        this.doReconnect = true

        // every 3 seconds, check if a reconnect is needed
        setInterval(() => {
            if (this.doReconnect) {
                this.doReconnect = false
                // connect does not return until a disconnect
                this.connect(zwConfig)
                    .then(() => {
                        log.info("connectLoop: connect successful")
                    })
                    .catch((e) => {
                        log.error("connectLoop: no connection with controller",e);
                        //retry
                        // fix exception
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
        const deviceID: string = this.homeID + "." + nodeID.toString();
        return deviceID
    }

    // return the node for the given deviceID or undefined if the deviceID is not known
    // @param deviceID as provided by getDeviceID
    getNodeByDeviceID(deviceID: string): ZWaveNode | undefined {
        const parts = deviceID.split(".")
        if (parts.length != 2) {
            return undefined
        }
        const nodeID = Number(parts[1])
        const node = this.driver.controller.nodes.get(nodeID)
        return node
    }

    // Driver is ready.
    handleDriverReady() {

        /*
          Now the controller interview is complete. This means we know which nodes
          are included in the network, but they might not be ready yet.
          The node interview will continue in the background.
          */
        const ctl = this.driver.controller;
        // homeID is ready after the controller interview
        // this.homeID = ctl.homeId ? ctl.homeId.toString(16).toUpperCase() : "n/a";

        log.info("Cache Dir: ", this.driver.cacheDir);
        log.info("Home ID:   ", this.driver.controller.homeId?.toString(16));

        ctl.nodes.forEach((node:ZWaveNode) => {
            // Subscribe to each node to catch its ready event.
            this.addNode(node);
        });

        // controller emitted events
        this.driver.controller.on("exclusion failed", () => {
            log.info("exclusion has failed");
        });
        this.driver.controller.on("exclusion started", () => {
            log.info("exclusion has started");
        });
        this.driver.controller.on("exclusion stopped", () => {
            log.info("exclusion has stopped");
        });

        this.driver.controller.on("rebuild routes progress",
            (progress: ReadonlyMap<number, RebuildRoutesStatus>) => {
                log.info("rebuild routes progress:", progress);
            });
        this.driver.controller.on("rebuild routes done", () => {
            log.info("rebuild routes done");
        });

        this.driver.controller.on("inclusion failed", () => {
            log.info("inclusion has failed");
        });
        // for zwave-js-12
        // this.driver.controller.on("inclusion started", (secure: boolean) => {
        //     log.info("inclusion has started. secure=", secure);
        // });
        // for zwave-js-13 and up
        this.driver.controller.on("inclusion started", (strategy: InclusionStrategy) => {
            log.info("ZWinclusion has started. strategy=%v", strategy);
        });
        this.driver.controller.on("inclusion stopped", () => {
            log.info("ZWinclusion has stopped");
        });

        this.driver.controller.on("node added", (node: ZWaveNode, result: InclusionResult) => {
            // a node was added to the network and initial setup completed
            // the node is not yet ready to be used until after the node interview.
            log.info(`ZWnode added: nodeId=${node.id} lowSecurity=${result.lowSecurity}`)
            this.setupNode(node);
        });
        this.driver.controller.on("node found", (node: FoundNode) => {
            // At this point, the initial setup and the node interview is still pending, so the node is not yet operational.
            log.info(`new found: nodeId=${node.id}`)
        });
        this.driver.controller.on("node removed", (node: ZWaveNode, reason: RemoveNodeReason) => {
            log.info(`ZWnode removed: id=${node.id}, reason=${reason}`);
        });

    }


    // return the homeID of the driver
    // This returns the homeID as hex string
    get homeID(): string {
        const hid = this.driver.controller.homeId;
        const homeIDStr = hid?.toString(16).toUpperCase() || "n/a"
        return homeIDStr
    }

    // setup a new node after it is added and listen for its events
    setupNode(node: ZWaveNode) {
        console.log("setting up node", node.id)
        // first time publish node TD and value map
        // this.onNodeUpdate?.(node);

        node.on("alive", (node: ZWaveNode, oldStatus: NodeStatus) => {
            log.info(`ZWNode ${node.id}: is alive`);
            if (node.status != oldStatus) {
                this.onStateUpdate(node, "alive")
            }
        });
        node.on("dead", (node: ZWaveNode, oldStatus: NodeStatus) => {
            log.info(`ZWNode ${node.id}: is dead`);
            if (node.status != oldStatus) {
                this.onStateUpdate(node, "dead")
            }
        });

        node.on("interview completed", (node: ZWaveNode) => {
            log.info(`ZWNode ${node.id}: interview completed`);
            // reload the TD
            this.onNodeUpdate(node)
        });
        node.on("interview failed", (node: ZWaveNode) => {
            log.info(`ZWNode ${node.id}: interview failed`);
            this.onStateUpdate(node, "interview failed")
        });
        node.on("interview started", (node: ZWaveNode) => {
            log.info(`ZWNode ${node.id}: interview started`);
            this.onStateUpdate(node, "interview started")
        });

        node.on("metadata updated", (node: ZWaveNode, args: ZWaveNodeMetadataUpdatedArgs) => {
            // FIXME: this is invoked even when metadata isn't updated. What to do?
            // this.onNodeUpdate(node)
            // let newValue = node.getValue(args)
            const newValue = getVidValue(node,args)
            log.info(`ZWNode ${node.id} value metadata updated`,
                "property=", args.property,
                "propertyKeyName=", args.propertyKeyName,
                "newValue=", newValue,
            );
            // FIXME: this causes duplicate events
            // this.onValueUpdate(node, args, newValue)
        });
        // node.on("notification", (endpoint: Endpoint, cc: CommandClasses, args:any) => {
        //     log.info(`Node ${endpoint.nodeId} Notification: CC=${cc}, args=${args}`)
        //     // TODO: what/when is this notification providing?
        // });
        node.on("ready", (node2:any) => {
            // log.info("--- Node", node2.id, "is ready. Updated the node. status=",node2.status);
            // handle in interview completed
            // this.onNodeUpdate(node);
        });

        node.on("sleep", (node: ZWaveNode) => {
            log.info(`ZWNode ${node.id}: is sleeping`);
            this.onStateUpdate(node, "sleeping")
        });

        // node.on("statistics updated", (node: ZWaveNode, args: NodeStatistics) => {
        //     // log.info("Node ", node.id, " stats updated: args=", args);
        // });

        node.on("value added", (node: ZWaveNode, args: ZWaveNodeValueAddedArgs) => {
            log.info(`ZWNode ${node.id}: value added: zw-property-name=${args.propertyName}, value=${args.newValue}`);
            // FIXME: only update the node if the value is new
            // refreshvalues triggers this a lot. we don't want to send a new TD for each value
            // this.onNodeUpdate(node)
            this.onValueUpdate(node, args, args.newValue)
        });

        node.on("value notification", (node: ZWaveNode, vid: ZWaveNodeValueNotificationArgs) => {
            log.info(`ZWNode ${node.id}, value notification: propName=${vid.propertyName}, value=${vid.value}`);
            this.onValueUpdate(node, vid, vid.value)
        });

        node.on("value removed", (node: ZWaveNode, args: ZWaveNodeValueRemovedArgs) => {
            log.info("Node ", node.id, " value removed for ", args.propertyName, ":", args.prevValue);
            this.onValueUpdate(node, args, undefined)
        });

        node.on("value updated", (node: ZWaveNode, args: ZWaveNodeValueUpdatedArgs) => {
            // log.debug("Node ", node.id, " value updated: args=", args);
            // convert enums
            const newVidValue = getVidValue(node,args)
            this.onValueUpdate(node, args, newVidValue)
            // this.onValueUpdate(node, args, args.newValue)
        });

        node.on("wake up", (node: ZWaveNode) => {
            log.info(`Node ${node.id}: wake up`);
            this.onStateUpdate(node, "awake")
        });
    }
}

// convert the string value to a type suitable for the vid
// If the vid is an enum then convert the enum label to its internal value
// export function stringToValue(value:string, node: ZWaveNode, vid:ValueID ):any {
//     let vidMeta = node.getValueMetadata(vid)
//     if (!vidMeta) {
//         return undefined
//     }
//     let vl = value.toLowerCase()
//     switch (vidMeta.type) {
//         case "string": return value;
//         case "boolean":
//             if (value === "" || vl ==="false" || value === "0" || vl==="disabled") {
//                 return false
//             }
//             return true
//         case "number":
//         case "duration":
//         case "color":
//             let numMeta = vidMeta as ValueMetadataNumeric;
//             let dataToSet: number|undefined
//             if (numMeta && numMeta.states) {
//                 dataToSet = getEnumFromMemberName(numMeta.states, value)
//             } else {
//                 dataToSet = parseInt(value,10)
//             }
//             if (isNaN(dataToSet as number)) {
//                 dataToSet = undefined
//             }
//             return dataToSet
//         case "boolean[]":
//         case "number[]":
//         case "string[]":
//             // TODO: support of arrays
//             log.error("getNewValueOfType data type '"+vidMeta.type+"' is not supported")
//     }
//     return value
// }


// Revert the enum name to its number
// This is the opposite of getEnumMemberName
export function getEnumFromMemberName(enumeration: Record<number, string>, name: string): number | undefined {
    for (const key in enumeration) {
        const val = enumeration[key]
        if (val?.toLowerCase() == name.toLowerCase()) {
            return Number(key)
        }
    }
    // in case the enum is optional and the name is a number already
    return Number(name)
}


// return the map of discovered ZWave nodes
// getNodes(): ReadonlyThrowingMap<number, ZWaveNode> {
//   return this.driver.controller.nodes
// }

// getVidValue returns the value of the node VID
// If the vid is an enum then return the text representation of the internal value,
// otherwise return the native value.
// Intended to transparently deal with enums.
// See also SetValue which does the reverse
export function getVidValue(node: ZWaveNode, vid:ValueID):any {
    const vidMeta = node.getValueMetadata(vid)
    let value = node.getValue(vid)
    if (vidMeta.type === "number") {
        const vmn = vidMeta as ValueMetadataNumeric;
        // if this vid has enum values then convert the value to its numeric equivalent
        if (vmn.states) {
            value = vmn.states[value as number]
        }
    }
    return value
}
