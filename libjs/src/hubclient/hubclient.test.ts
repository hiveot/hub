// mqtt and nats transport testing

import * as process from "process";
import {DeliveryStatus, IHubClient} from './IHubClient';
import { ThingMessage } from "../things/ThingMessage";
import {ConnectToHub} from "@hivelib/hubclient/ConnectToHub";

let hc: IHubClient
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
    let hc =await ConnectToHub(testURL, testClient, caCertPEM)

    hc.setActionHandler((tv: ThingMessage):DeliveryStatus => {
        console.log("Received action: " + tv.key)
        let stat = new DeliveryStatus()
        stat.Completed(tv)
        return stat
    })

    hc.setEventHandler((tv: ThingMessage) => {
        console.log("onEvent: " + tv.key + ": " + tv.data)
    })

    await hc.connectWithPassword(testPass)

    // subscribe to all events
    hc.subscribe("", "")

    // publish an action request 
    let stat = await hc.pubAction("thing1", "action1", "1")
    if (stat.reply != "action") {
        throw ("unexpected action reply")
    }

    // publish a config request 
    stat = await hc.pubConfig("thing1", "prop1", "10")
    if (stat.reply != "true") {
        throw ("unexpected config reply")
    }

    // rpc request
    try {
        let reply = await hc.rpc( "cap1", "method1", "data")
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