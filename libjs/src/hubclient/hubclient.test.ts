// mqtt and nats transport testing

import process from "node:process";
import * as tslog from 'tslog';
import {DeliveryProgress, DeliveryStatus} from './IAgentClient';
import {ThingMessage} from "../things/ThingMessage";
import {ConnectToHub} from "@hivelib/hubclient/ConnectToHub";
import {EventTypeDeliveryUpdate, MessageTypeAction} from "@hivelib/api/vocab/ht-vocab";

const log = new tslog.Logger({name: "HCTest"})

process.on("uncaughtException", (err: any) => {
    log.error("uncaughtException", err)
})

// URL of a functional runtime
let caCertPEM = ""
const baseURL = "https://localhost:8444"
const testSvcID = "testsvc"
const testSvcPass = "testpass"
const testClientID = "test"
const testPass = "test22"

// test connecting with password and token refresh
async function test1() {
    let token: string

    //running instance
    let hc = await ConnectToHub(baseURL, testClientID, caCertPEM, true)
    try {
        token = await hc.connectWithPassword(testPass)
        if (token == "") {
            throw("missing token")
        }
        token = await hc.refreshToken()
    } catch (err) {
        log.error("Failed connecting to server: " + err)
        return
    }
    //round 2, read digitwin
    try {

    } catch(e) {
        log.error("test1: Failed: " + e)
    }
    hc.disconnect()
}

// test add a service and publish events
async function test2() {
    let lastMsg = ""
    let token: string

    //running instance
    let hc = await ConnectToHub(baseURL, testSvcID, caCertPEM, true)
    try {
        token = await hc.connectWithPassword(testSvcPass)
    } catch(e) {
        log.error("test2",e)
        throw(e)
    }

    let stat = await hc.pubEvent("thing1", "event1", "hello world")
    if (stat.error) {
        log.error("pubEvent failed:",stat.error)
    } else if (stat.progress != DeliveryProgress.DeliveryCompleted) {
        log.error("pubEvent status not 'completed': progress=", stat.progress)
    }
    hc.disconnect()
}

// test reading directory
async function test3() {
    let lastMsg = ""
    let token: string
    const thingID = "dtw:agent1:thing1"

    // todo have agent publish TD of thing1 and listen for actions

    //running instance
    let hc =await ConnectToHub(baseURL, testClientID, caCertPEM,true)
    try {
        token = await hc.connectWithPassword(testPass)

        hc.setMessageHandler((tv: ThingMessage):DeliveryStatus => {
            log.info("Received message: type="+tv.messageType+"; key=" + tv.name)
            let stat = new DeliveryStatus()
            stat.completed(tv)
            return stat
        })
        token = await hc.refreshToken()
    } catch (err) {
        log.error("Failed connecting to server: "+err)
        return
    }

    // subscribe to all events
    try {
        await hc.subscribe("", "")

        // publish an action request
        let stat = await hc.pubAction(thingID, "action1", "1")
        if (stat.error != "") {
            throw ("pubAction failed: " + stat.error)
        }

        // publish a config request
        stat = await hc.pubProperty("thing1", "prop1", "10")
        if (stat.error != "") {
            throw ("pubConfig failed: " + stat)
        }
    } catch (err) {
        log.error("error subscribing and publishing: ", err)
    }
    // rpc request
    try {
        let reply = await hc.rpc( "cap1", "method1", "data")
        if (reply != "rpc") {
            throw ("unexpected rpc reply")
        }
    } catch (e) {
        log.error("rpc error", e)
    }

    // await new Promise(resolve => setTimeout(resolve, 1200000));
    hc.disconnect()
}

// test sse by subscribing to events
async function test4() {
    let svcToken = ""
    let clToken = ""
    let ev1Count = 0
    let actionCount = 0
    let actionDelivery: DeliveryStatus|undefined

    // connect a service that sends events
    let hcSvc = await ConnectToHub(baseURL, testSvcID, caCertPEM, true)
    try {
        svcToken = await hcSvc.connectWithPassword(testSvcPass)
    } catch(e) {
        log.error("test4",e)
        throw(e)
    }

    // connect a client that listens for events
    let hcCl = await ConnectToHub(baseURL, testClientID, caCertPEM, true)
    try {
        clToken = await hcCl.connectWithPassword(testPass)
    } catch (err) {
        log.error("Failed connecting to server: " + err)
        return
    }

    //round 2, subscribe to events
    try {
        await hcCl.subscribe("dtw:testsvc:thing1","")
        // await hcCl.subscribe("","")
        hcCl.setMessageHandler((tm: ThingMessage):DeliveryStatus=>{
            let stat = new DeliveryStatus()
            if (tm.thingID == "dtw:testsvc:thing1") {
                log.info("Received event: "+tm.name+"; data="+tm.data)
                ev1Count++
            } else if (tm.name == EventTypeDeliveryUpdate) {
                // FIXME: why is data base64 encoded? => data type in golang was []byte; changed to string
                // let data = Buffer.from(tm.data,"base64").toString()
                actionDelivery = JSON.parse(tm.data)
            } else if (tm.messageType == MessageTypeAction) {
                actionCount++
                stat.reply = "success"
            }
            stat.completed(tm)
            return stat
        })
    } catch(e) {
        log.error("test1: Failed: " + e)
    }

    // time for background to start listening
    await new Promise(resolve => setTimeout(resolve, 100));

    // round 3, send a test event
    let stat = await hcSvc.pubEvent("thing1", "event1", "hello world")
    if (stat.error) {
        log.error("failed publishing event: "+stat.error)
    }

    // round 4, send an action to the digitwin thing of the test service
    let dtwThing1ID = "dtw:"+testSvcID+":thing1"
    let stat2 = await hcCl.pubAction(dtwThing1ID,"action1", "how are you")
    if (stat2.error) {
        log.error("failed publishing action: "+stat2.error)
    } else if (stat2.progress != DeliveryProgress.DeliveryToAgent) {
        log.error("unexpected reply: "+stat2.progress)
    }

    // wait for events
    await new Promise(resolve => setTimeout(resolve, 1000));

    if (ev1Count != 1) {
        log.error("received " + ev1Count + " events. Expected 1")
    } else {
        log.info("test4 event success. Received an event")
    }
    if (actionCount != 1) {
        log.error("received " + actionCount + " actions. Expected 1")
    } else if (!actionDelivery || actionDelivery.progress != DeliveryProgress.DeliveryCompleted) {
        log.error("test4 action sent but missing delivery confirmation")
    } else {
        log.info("test4 action success. Received an action confirmation")
    }
    hcSvc.disconnect()
    hcCl.disconnect()
}

// jest isn't working with tsx yet. Once it does then lets change the tests
// describe("test connect", () => {
//     it('should connect', async () => {
//         //running instance
//         const testClientID = "test"
//         const testPass = "testpass"
//         let caCertPEM = ""
//         let hc = NewHubClient(testURL, testClientID, caCertPEM, core)
//         await hc.connectWithPassword(testPass)

//     })
// })

// These tests require a running test environment
// test1()
//  test2()
 // test3()
test4()
