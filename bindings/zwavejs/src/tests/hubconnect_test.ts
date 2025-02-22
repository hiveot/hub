// mqtt and nats transport testing

import process from "node:process";
import {connect} from 'node:http2';
import {HttpSSEClient} from "@hivelib/messaging/httpclient/HttpSSEClient.js";
import {ConnectToHub} from "@hivelib/messaging/ConnectToHub";
import ky from 'ky';
import {getlogger} from "@zwavejs/getLogger";

process.on("uncaughtException", (err: Error) => {
    log.error("uncaughtException", err)
})

// test setup using a running server
const testURL = "https://localhost:8444"
const testClient = "test"
const testPass = "test22"
let caCertPEM = ""

const log = getlogger()

// test connect with password and refresh token
async function test1() {
    //
    // process.on("uncaughtException", (err: Error) => {
    //     log.error("uncaughtException", err)
    // })
log.info("connecting to "+testURL)
    let hc = await ConnectToHub( testURL,testClient, caCertPEM,true)
    let token = await hc.connectWithPassword(testPass)

    log.info("publishing hello world")
    hc.pubEvent("testthing", "event1", "hello world")
    hc.disconnect()
}

// test list directory using actions
async function test2() {

    let hc = await ConnectToHub( testURL,testClient, caCertPEM, true)
    let token = await hc.connectWithPassword(testPass)

    // sending read directory request
    let resp = await hc.invokeAction(
        "dtw:digitwin:directory", "ReadTDs", '{"limit":10}')

    if (resp.correlationID == "") {
        log.error("readTDs didn't return a correlationID")
    } else if (!resp.output)  {
        log.error("readTDs didn't return any data")
    } else {
        let tdList = JSON.parse(resp.output)
        if (tdList.length == 0) {
            log.error("Didnt receive any TDs")
        } else {
            log.info("Received "+tdList.length+" TDs")
        }
    }
    hc.disconnect()
}

// test list directory using rest api
async function test3() {

    let fullURL= testURL+"/authn/login"
    let hc = new HttpSSEClient(testURL, testClient, caCertPEM, true)
    let authToken = await hc.connectWithPassword(testPass)

    let client = connect(testURL, {
        "rejectUnauthorized": false,
    });
    client.on('error',(err)=>{
        console.error(err);
    })

    let body = '{"login":"test", "password":"test22"}';
    const req = client.request({
        origin: testURL,
        authorization: "bearer "+authToken,
        ':path': '/things',
        ":method": "GET",
        "content-type": "application/json",
    })
    req.setEncoding('utf8');
    let data = '';
    req.on('data', (chunk) => {
        data += chunk;
    });
    req.on('end', () => {
        console.log(`\n${data}`);
        client.close();
        hc.disconnect()
    });
    req.end();
}



async function waitForSignal() {

    //--- Step 4: Wait for  SIGINT or SIGTERM signal to stop
    log.info("Ready. Waiting for signal to terminate")
    try {
        for (const signal of ["SIGINT", "SIGTERM"]) {

            process.on(signal, async () => {
                log.info("signal received!: ", signal)
                // await hc.disconnect();
                // exit(0);
            });
        }
    } catch (e) {
        log.error("Error: ", e)
    }

}


test1()
// test2()
// test3()