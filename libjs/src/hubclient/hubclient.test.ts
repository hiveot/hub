// mqtt and nats transport testing

import process from "node:process";
import * as tslog from 'tslog';
import {DeliveryStatus, IHubClient} from './IHubClient';
import { ThingMessage } from "../things/ThingMessage";
import {ConnectToHub} from "@hivelib/hubclient/ConnectToHub";

const log = new tslog.Logger({name: "HCTest"})

process.on("uncaughtException", (err: any) => {
    log.error("uncaughtException", err)
})

// URL of a functional runtime
let caCertPEM = ""
const baseURL = "https://localhost:8444"
const testSvc = "testsvc"
const testSvcPass = "testpass"
const testClient = "test"
const testPass = "test22"

// test connecting with password and token refresh
async function test1() {
    let token: string

    //running instance
    let hc = await ConnectToHub(baseURL, testClient, caCertPEM, true)
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
    let hc = await ConnectToHub(baseURL, testSvc, caCertPEM, true)
    try {
        token = await hc.connectWithPassword(testSvcPass)
    } catch(e) {
        log.error("test2",e)
        throw(e)
    }

    let stat = await hc.pubEvent("thing1", "event1", "hello world")
    if (stat.error) {
        log.error("pubEvent failed:",stat.error)
    } else if (stat.progress != "completed") {
        log.error("pubEvent status not delivered but:", stat.progress)
    }
    hc.disconnect()
}

// test reading directory
async function test3() {
    let lastMsg = ""
    let token: string

    //running instance
    let hc =await ConnectToHub(baseURL, testClient, caCertPEM,true)
    try {
        token = await hc.connectWithPassword(testPass)

        hc.setActionHandler((tv: ThingMessage):DeliveryStatus => {
            log.info("Received action: " + tv.key)
            let stat = new DeliveryStatus()
            stat.Completed(tv)
            return stat
        })

        hc.setEventHandler((tv: ThingMessage) => {
            log.info("onEvent: " + tv.key + ": " + tv.data)
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
        let stat = await hc.pubAction("thing1", "action1", "1")
        if (stat.error != "") {
            throw ("pubAction failed: " + stat.error)
        }

        // publish a config request
        stat = await hc.pubConfig("thing1", "prop1", "10")
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

    // connect a service that sends events
    let hcSvc = await ConnectToHub(baseURL, testSvc, caCertPEM, true)
    try {
        svcToken = await hcSvc.connectWithPassword(testSvcPass)
    } catch(e) {
        log.error("test4",e)
        throw(e)
    }

    // connect a client that listens for events
    let hcCl = await ConnectToHub(baseURL, testClient, caCertPEM, true)
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
        hcCl.setEventHandler((tm: ThingMessage)=>{
            if (tm.thingID == "dtw:testsvc:thing1") {
                // let data = JSON.parse(tm.data)
                // FIXME: why is data base64 encoded?
                let data = Buffer.from(tm.data,"base64").toString()
                log.info("Received event: "+tm.key+"; data="+data)
                ev1Count++
            }
        })
        // set an action handler for the service
        hcSvc.setActionHandler((tm:ThingMessage):DeliveryStatus=>{
            let stat = new DeliveryStatus()
            stat.Completed(tm)
            stat.reply = "success"
            actionCount++
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

    // round 4, send an action
    let stat2 = await hcCl.pubAction(testSvc,"action1", "how are you")
    if (stat2.error) {
        log.error("failed publishing action: "+stat2.error)
    }

    // wait 1 minute for events
    await new Promise(resolve => setTimeout(resolve, 600000));

    if (ev1Count != 1) {
        log.error("received " + ev1Count + " events. Expected 1")
    } else {
        log.info("test4 success. Received an event")
    }
    if (actionCount != 1) {
        log.error("received " + actionCount + " actions. Expected 1")
    }
    hcSvc.disconnect()
    hcCl.disconnect()
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

// test1()
// test2()
// test3()
test4()
