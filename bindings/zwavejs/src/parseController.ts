import type {ZWaveController, ZWaveNode} from "npm:zwave-js";
import type {TD} from "@hivelib/wot/TD.ts";
import {DataSchema} from "@hivelib/wot/dataSchema.ts";
import {WoTDataTypeNone, WoTDataTypeString} from "@hivelib/api/vocab/vocab.js";
import {RFRegion} from "npm:zwave-js";

// parseController adds controller actions and attributes to the Thing TD
export function parseController(td: TD, ctl: ZWaveController) {

    if (ctl.rfRegion) {
        td.AddProperty("rfRegion",  "RF Region","Geographic region for RF emission rules",WoTDataTypeString)
            .SetAsEnum(RFRegion)
            .SetAsConfiguration()
            .SetDescription("RF Region the controller is set to")
    }

    // controller events. Note these must match the controller event handler
    td.AddEvent("healNetworkState",  "Heal Network Progress", undefined,
        new DataSchema({title: "Heal State", type: WoTDataTypeString}))
    td.AddEvent("inclusionState",  "Node Inclusion Progress", undefined,
        new DataSchema({title: "Inclusion State", type: WoTDataTypeString}))
    td.AddEvent("nodeAdded", "Node Added", undefined,
        new DataSchema({title: "ThingID", type: WoTDataTypeString}))
    td.AddEvent("nodeRemoved",  "Node Removed", undefined,
        new DataSchema({title: "ThingID", type: WoTDataTypeString}))

    // controller network actions
    td.AddAction("beginInclusion", "Start add node process",
        "Start the inclusion process for new nodes. Prefer S2 security if supported")
    td.AddAction("stopInclusion",  "Stop add node process")
    td.AddAction("beginExclusion", "Start node removal process")
    td.AddAction("stopExclusion",  "Stop node removal process")
    td.AddAction("beginHealingNetwork", "Start heal network process",
        "Start healing the network routes. This can take a long time and slow things down.")
    td.AddAction("stopHealingNetwork", "Stop the ongoing healing process")

    // controller node actions
    td.AddAction("getNodeNeighbors",  "Update Neighbors",
        "Request update to a node's neighbor list",
        new DataSchema({title: "ThingID", type: WoTDataTypeString})
    )
    td.AddAction("healNode", "Heal the node",
        "Heal the node and update its neighbor list",
        new DataSchema({title: "ThingID", type: WoTDataTypeString})
    )
    td.AddAction("removeFailedNode", "Remove failed node",
        "Remove a failed node from the network",
        new DataSchema({title: "ThingID", type: WoTDataTypeString})
    )
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
        for (let [param, pi] of ownNode.deviceConfig.paramInformation) {
            let propID = `${pi.parameterNumber} ${pi.label}`
            let title = propID
            let dataType = WoTDataTypeNone
            // FIXME: Include Data schema handle dataType, minValue, maxValue, options, unit, readOnly, writeOnly, unsigned
            let prop = td.AddProperty(propID, dataType, title,"")
            prop.readOnly = false
        }
    }

}
