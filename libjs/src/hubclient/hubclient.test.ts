// mqtt and nats transport testing

import { MqttTransport } from "./transports/mqtttransport/MqttTransport";
import { env, exit } from "process";
import * as process from "process";
import { HubClient, NewHubClient } from "@hivelib/hubclient/httpclient/HubClient";
import { IHubClient } from './IHubClient';
import { ThingMessage } from "../things/ThingMessage";

let hc: HubClient
let tp: IHubClient
const testURL = "https://127.0.0.1:9883"

async function test1() {
    let lastMsg = ""
    process.on("uncaughtException", (err: any) => {
        console.error("uncaughtException", err)
    })

    const testClient = "test"
    const testPass = "testpass"
    let caCertPEM = ""
    //running instance
    let hc = NewHubClient(testURL, testClient, caCertPEM)

    hc.setActionHandler((tv: ThingMessage) => {
        console.log("Received action: " + tv.key)
        return "action"
    })

    hc.setEventHandler((tv: ThingMessage) => {
        console.log("onEvent: " + tv.key + ": " + tv.data)
    })

    hc.setConfigHandler((tv: ThingMessage) => {
        console.log("onConfig: " + tv.key)
        return true
    })

    await hc.connectWithPassword(testPass)

    // subscribe to all events
    hc.subscribe("", "", "")

    // publish an action request 
    let reply = await hc.pubAction(testClient, "thing1", "action1", "1")
    if (reply != "action") {
        throw ("unexpected action reply")
    }

    // publish a config request 
    let breply = await hc.pubConfig(testClient, "thing1", "prop1", "10")
    if (breply != true) {
        throw ("unexpected config reply")
    }

    // rpc request
    try {
        reply = await hc.pubRPCRequest(testClient, "cap1", "method1", "data")
        if (reply != "rpc") {
            throw ("unexpected rpc reply")
        }
    } catch (e) {
        console.log("timeout works")
    }

    await new Promise(resolve => setTimeout(resolve, 1200000));
    hc.disconnect()
}


// jest isn't working with tsx yet. Once it does then lets change the tests
// describe("test connect", () => {
//     it('should connect', async () => {
//         //running instance
//         const testClient = "test"
//         const testPass = "testpass"
//         let caCertPEM = ""
//         let hc = NewHubClient(testURL, testClient, caCertPEM, core)
//         await hc.connectWithPassword(testPass)

//     })
// })

test1()