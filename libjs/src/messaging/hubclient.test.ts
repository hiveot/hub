import process from "node:process";
import * as tslog from 'tslog';
import {ConnectToHub} from "@hivelib/messaging/ConnectToHub";
import { OpInvokeAction} from "@hivelib/api/vocab/vocab.js";
import {RequestMessage, ResponseMessage, StatusCompleted, StatusPending} from "@hivelib/messaging/Messages";

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

        hc.setRequestHandler((req: RequestMessage):ResponseMessage => {
            log.info("Received message: type="+req.operation+"; key=" + req.name)
            let resp = req.createResponse("")
            return resp
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
        let stat = await hc.invokeAction(thingID, "action1", "1")
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
        let reply = await hc.rpc( OpInvokeAction, "cap1", "method1", "data")
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
    let actionDelivery: ResponseMessage | undefined

    // connect a service that sends events
    let agc = await ConnectToHub(baseURL, testSvcID, caCertPEM, true)
    try {
        svcToken = await agc.connectWithPassword(testSvcPass)
    } catch (e) {
        log.error("test4", e)
        throw (e)
    }

    // connect a client that listens for events
    let clc = await ConnectToHub(baseURL, testClientID, caCertPEM, true)
    try {
        clToken = await clc.connectWithPassword(testPass)
    } catch (err) {
        log.error("Failed connecting to server: " + err)
        return
    }

    //round 2, subscribe to events
    try {
        await clc.subscribe("dtw:testsvc:thing1", "")
        // await hcCl.subscribe("","")
        clc.setRequestHandler((req: RequestMessage): ResponseMessage => {
            actionCount++
            let resp = req.createResponse("success")
            return resp
        })

        clc.setResponseHandler((notif: ResponseMessage) => {
            if (notif.thingID == "dtw:testsvc:thing1") {
                log.info("Received event: " + notif.name + "; data=" + notif.output)
                ev1Count++
            }
        })
        clc.setResponseHandler((resp: ResponseMessage) => {
            actionDelivery = JSON.parse(resp.output)
        })

    } catch (e) {
        log.error("test1: Failed: " + e)
    }


        // time for background to start listening
    await new Promise(resolve => setTimeout(resolve, 100));

    // round 3, send a test event
    // FIXME: publish a TD for thing1 to create a digitwin
    try {
        await agc.pubEvent("thing1", "event1", "hello world")
    } catch (e) {
        console.error("pubevent failed",e)
    }
    // round 4, send an action to the digitwin thing of the test service
    try {
        agc.setRequestHandler((req: RequestMessage): ResponseMessage => {
            let resp: ResponseMessage
            // agents receive the thingID without prefix
            if (req.thingID == "thing1") {
                resp = req.createResponse("success!")
                console.info("success!")
            } else {
                resp = req.createResponse("", Error("not receiving action"))
                console.error("not receiving action")
            }
            return resp
        })

        let dtwThing1ID = "dtw:" + testSvcID + ":thing1"
        let resp2 = await clc.invokeAction(dtwThing1ID, "action1",  "how are you")
        if (resp2.error) {
            log.error("failed publishing action: " + resp2.error)
        } else if (resp2.status != StatusPending) {
            log.error("unexpected status: " + resp2.status)
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
        } else if (!actionDelivery || actionDelivery.status != StatusCompleted) {
            log.error("test4 action sent but missing delivery confirmation")
        } else {
            log.info("test4 action success. Received an action confirmation")
        }
        clc.disconnect()
        agc.disconnect()
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