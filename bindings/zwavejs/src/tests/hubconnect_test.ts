// mqtt and nats transport testing

import { NewHubClient } from "@hivelib/hubclient/HubClient";
import process from "node:process";
import * as tslog from 'tslog';

const log = new tslog.Logger({ name: "TTT" })

async function test1() {

    process.on("uncaughtException", (err: Error) => {
        log.error("uncaughtException", err)
    })

    const url = "mqtts://127.0.0.1:8883"
    const testClient = "test"
    const testPass = "testpass"
    let caCertPEM = ""
    //running instance
    // tp = new MqttTransport("mqtts://127.0.0.1:8883", testClient, caCertPEM)
    let hc = NewHubClient(url, testClient, caCertPEM)
    await hc.connectWithPassword(testPass)

    log.info("publishing hello world")
    await hc.pubEvent("testthing", "event1", "hello world")

    // tp.sub("event/test/#",(ev)=>{
    //     log.log("rx ev",ev)
    // })

    log.info("publishing hello world2")
    await hc.pubEvent("testthing", "event2", "hello world2")

    await waitForSignal()

    await new Promise(resolve => setTimeout(resolve, 5000));

    log.info("Disconnecting...")
    hc.disconnect()
}

async function waitForSignal() {

    //--- Step 4: Wait for  SIGINT or SIGTERM signal to stop
    log.info("Ready. Waiting for signal to terminate")
    try {
        for (const signal of ["SIGINT", "SIGTERM"]) {

            await process.on(signal, async () => {
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