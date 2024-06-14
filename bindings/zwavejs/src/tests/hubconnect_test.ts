// mqtt and nats transport testing

import process from "node:process";
import * as tslog from 'tslog';
import {connect} from 'node:http2';
import {HttpSSEClient} from "@hivelib/hubclient/httpclient/HttpSSEClient.js";
import {ConnectToHub} from "@hivelib/hubclient/ConnectToHub";
import ky from 'ky';

process.on("uncaughtException", (err: Error) => {
    log.error("uncaughtException", err)
})

// test server setup
const testURL = "https://localhost:8444"
const testClient = "test"
const testPass = "test22"
let caCertPEM = ""

const log = new tslog.Logger({name: "TTT"})

// test connect with password and refresh token
async function test1() {
    //
    // process.on("uncaughtException", (err: Error) => {
    //     log.error("uncaughtException", err)
    // })

    let hc = await ConnectToHub( testURL,testClient, caCertPEM,true)
    let token = await hc.connectWithPassword(testPass)

    log.info("publishing hello world")
    let stat = await hc.pubEvent("testthing", "event1", "hello world")
    if (stat.messageID == "") {
        log.error("pubEvent didn't return a messageID")
    }
    hc.disconnect()
}

// test list directory using actions
async function test2() {

    let hc = await ConnectToHub( testURL,testClient, caCertPEM, true)
    let token = await hc.connectWithPassword(testPass)

    // sending read directory request
    let stat = await hc.pubAction(
        "dtw:digitwin:directory", "readTDs", '{"limit":10}')

    if (stat.messageID == "") {
        log.error("readTDs didn't return a messageID")
    } else if (!stat.reply)  {
        log.error("readTDs didn't return any data")
    } else {
        let tdList = JSON.parse(stat.reply)
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

    let body = '{"clientID":"test", "password":"test22"}';
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