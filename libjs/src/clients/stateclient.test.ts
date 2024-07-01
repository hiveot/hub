import { HttpSSEClient } from "@hivelib/hubclient/httpclient/HttpSSEClient"
import { IHubClient } from "@hivelib/hubclient/IHubClient"
import { StateClient } from "./stateclient"

let hc: IHubClient
let tp: IHubClient
const testURL = "https://127.0.0.1:"+9883


process.on("uncaughtException", (err: any) => {
    console.error("uncaughtException", err)
})

async function connect(): Promise<IHubClient> {
    // the server must have a test client 
    const clientID = "test"
    const testPass = "testpass"
    let caCertPEM = ""
    //running instance
    let hc = new HttpSSEClient(testURL, clientID, caCertPEM, true)
    await hc.connectWithPassword(testPass)
    return hc
}


async function testSet() {
    let testKey = "key1"
    let testText = "this is test text"

    let hc = await connect()
    let cl = new StateClient(hc)

    await cl.Set(testKey, testText)

    let resp = await cl.Get(testKey)
    if (resp.value != testText) {
        throw ("get result doesn't match set")
    }

    let multiple: { [index: string]: string } = {}
    multiple["key1"] = "val1"
    multiple["key2"] = "val2"

    await cl.SetMultiple(multiple)
    let resp2 = await cl.GetMultiple(["key1", "key2"])
    if (resp2["key1"] != "val1" || resp2["key2"] != "val2") {
        throw ("get multiple doesn't provide multiple results")
    }
}


testSet()
