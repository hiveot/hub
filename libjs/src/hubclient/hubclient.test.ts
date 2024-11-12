import process from "node:process";
import * as tslog from 'tslog';
import {RequestProgress} from './RequestProgress';
import {ThingMessage} from "../things/ThingMessage";
import {ConnectToHub} from "@hivelib/hubclient/ConnectToHub";
import {MessageTypeProgressUpdate, MessageTypeAction} from "@hivelib/api/vocab/vocab.js";
import {RequestCompleted, RequestDelivered} from "@hivelib/api/vocab/vocab.js";

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
        await hc.pubEvent("thing1", "event1", "hello world")
    } catch(e) {
        log.error("test2 - expected error",e)
    } finally {
        await hc.disconnect()
    }

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

        hc.setActionHandler((tm: ThingMessage):RequestProgress => {
            log.info("Received message: type="+tm.messageType+"; key=" + tm.name)
            let stat = new RequestProgress()
            stat.completed(tm)
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
        let requestID = "test3a"
        let stat = await hc.invokeAction(thingID, "action1",requestID, "1")
        if (stat.error != "") {
            throw ("pubAction failed: " + stat.error)
        }

        // publish a config request
        await hc.writeProperty("thing1", "prop1", "10")
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
    let actionDelivery: RequestProgress | undefined

    // connect a service that sends events
    let hcSvc = await ConnectToHub(baseURL, testSvcID, caCertPEM, true)
    try {
        svcToken = await hcSvc.connectWithPassword(testSvcPass)
    } catch (e) {
        log.error("test4", e)
        throw (e)
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
        await hcCl.subscribe("dtw:testsvc:thing1", "")
        // await hcCl.subscribe("","")
        hcCl.setActionHandler((tm: ThingMessage): RequestProgress => {
            let stat = new RequestProgress()
            if (tm.thingID == "dtw:testsvc:thing1") {
                log.info("Received event: " + tm.name + "; data=" + tm.data)
                ev1Count++
            } else if (tm.messageType == MessageTypeProgressUpdate) {
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
    } catch (e) {
        log.error("test1: Failed: " + e)
    }

    // time for background to start listening
    await new Promise(resolve => setTimeout(resolve, 100));

    // round 3, send a test event
    // FIXME: publish a TD for thing1 to create a digitwin
    try {
        await hcSvc.pubEvent("thing1", "event1", "hello world")
    } catch (e) {
        console.error("pubevent failed",e)
    }
    // round 4, send an action to the digitwin thing of the test service
    try {
        hcSvc.setActionHandler((msg: ThingMessage): RequestProgress => {
            let stat = new RequestProgress()
            // agents receive the thingID without prefix
            if (msg.thingID == "thing1") {
                console.info("success!")
            } else {
                console.error("not receiving action")
            }
            return stat
        })

        let dtwThing1ID = "dtw:" + testSvcID + ":thing1"
        let requestID = "test4a"
        let stat2 = await hcCl.invokeAction(dtwThing1ID, "action1", requestID, "how are you")
        if (stat2.error) {
            log.error("failed publishing action: " + stat2.error)
        } else if (stat2.progress != RequestDelivered) {
            log.error("unexpected reply: " + stat2.progress)
        }
    } catch (e) {
        console.log("invokeAction failed")
    }

    // wait for events
    try {
        await new Promise(resolve => setTimeout(resolve, 1000));

        if (ev1Count != 1) {
            log.error("received " + ev1Count + " events. Expected 1")
        } else {
            log.info("test4 event success. Received an event")
        }
        if (actionCount != 1) {
            log.error("received " + actionCount + " actions. Expected 1")
        } else if (!actionDelivery || actionDelivery.progress != RequestCompleted) {
            log.error("test4 action sent but missing delivery confirmation")
        } else {
            log.info("test4 action success. Received an action confirmation")
        }
        hcSvc.disconnect()
        hcCl.disconnect()
    } catch (e) {
        console.error("wait for events failed")
    }
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
// test1().then()
// test2().then()
//  test3()
test4()

console.log('done')