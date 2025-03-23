import type {ZWaveController, ZWaveNode} from "zwave-js";
import {RFRegion} from "zwave-js";

import TD from "../hivelib/wot/TD.ts";
import  DataSchema from "../hivelib/wot/dataSchema.ts";
import {WoTDataTypeNone, WoTDataTypeString} from "../hivelib/api/vocab/vocab.js";
import type ZWAPI from "./ZWAPI.ts";
import createNodeTD from "./createNodeTD.ts";


// Create the node of the controller.
// The controller has a few extra events and actions.
export default function createControllerTD(zwapi: ZWAPI, node: ZWaveNode, vidLogFD: number | undefined): TD {
    let tdi = createNodeTD(zwapi,node,vidLogFD,0)

    parseController(tdi, zwapi.driver.controller)
    return tdi
}

// parseController adds controller actions and attributes to the Thing TD
function parseController(tdi: TD, ctl: ZWaveController) {

    if (ctl.rfRegion) {
        tdi.AddProperty("rfRegion",  "RF Region","Geographic region for RF emission rules",WoTDataTypeString)
            .SetAsEnum(RFRegion)
            .SetAsConfiguration()
            .SetDescription("RF Region the controller is set to")
    }

    // controller events. Note these must match the controller event handler
    // tdi.AddEvent("healNetworkState",  "Heal Network Progress", undefined,
    //     new DataSchema({title: "Heal State", type: WoTDataTypeString}))
    // tdi.AddEvent("inclusionState",  "Return the node Inclusion Progress", undefined,
    //     new DataSchema({title: "Inclusion State", type: WoTDataTypeString}))
    tdi.AddEvent("nodeAdded", "Node Added", undefined,
        new DataSchema({title: "ThingID", type: WoTDataTypeString}))
    tdi.AddEvent("nodeRemoved",  "Node Removed", undefined,
        new DataSchema({title: "ThingID", type: WoTDataTypeString}))

    // controller network actions
    tdi.AddAction("beginInclusion", "Start add node process",
        "Start the inclusion process for new nodes. Prefer S2 security if supported")
    tdi.AddAction("beginExclusion", "Start remove node process")

    tdi.AddAction("beginRebuildingRoutes", "Start rebuilding routes",
        "Start healing the network routes. This can take a long time and slow things down.")


    tdi.AddAction("stopExclusion",  "Stop remove node process")
    tdi.AddAction("stopInclusion",  "Stop add node process")
    tdi.AddAction("stopRebuildingRoutes", "Stop rebuilding routes")

    // // controller node actions
    // tdi.AddAction("getNodeNeighbors",  "Update Neighbors",
    //     "Request update to a node's neighbor list",
    //     new DataSchema({title: "ThingID", type: WoTDataTypeString})
    // )
    // tdi.AddAction("healNode", "Heal the node",
    //     "Heal the node and update its neighbor list",
    //     new DataSchema({title: "ThingID", type: WoTDataTypeString})
    // )
    // tdi.AddAction("removeFailedNode", "Remove failed node",
    //     "Remove a failed node from the network",
    //     new DataSchema({title: "ThingID", type: WoTDataTypeString})
    // )
    // td.AddAction("replaceFailedNode", "Replace a failed node with another node (thingID, thingID)", DataType.String)

    // ctl.getBackgroundRSSI().then(value => {
    //     log.info("BackgroundRSSI:", value)
    // })

    // Apparently, controllers have no VIDs but can have configuration in deviceConfig.paramInformation
    let ownNode: ZWaveNode | undefined
    if (ctl.ownNodeId != undefined) {
        ownNode = ctl.nodes.get(ctl.ownNodeId)
    }
    if ((ownNode?.deviceConfig) && (ownNode.deviceConfig.paramInformation)) {
        for (const [_param, pi] of ownNode.deviceConfig.paramInformation) {
            const propID = `${pi.parameterNumber} ${pi.label}`
            const title = propID
            const dataType = WoTDataTypeNone
            // FIXME: Include Data schema handle dataType, minValue, maxValue, options, unit, readOnly, writeOnly, unsigned
            const prop = tdi.AddProperty(propID, dataType, title,"")
            prop.readOnly = false
        }
    }

}
