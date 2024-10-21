import * as tslog from 'tslog';
import {ConnectionHandler, ConnectionStatus, EventHandler} from "@hivelib/hubclient/IConsumerClient";
import {ThingMessage} from "@hivelib/things/ThingMessage";
import EventSource from 'eventsource'
import {DeliveryStatus} from "@hivelib/hubclient/DeliveryStatus";
import {
    MessageTypeAction,
    MessageTypeEvent,
    MessageTypeProgressUpdate,
    MessageTypeProperty
} from "@hivelib/api/vocab/vocab";

const hclog = new tslog.Logger()

function parseSSEEvent(e: MessageEvent) : ThingMessage {
    let tm = new ThingMessage()

    let sseEventID = e.lastEventId
    let parts = sseEventID.split("/")
    tm.thingID =parts[0]
    if (parts.length > 1) {
        tm.name = parts[1]
    }
    if (parts.length > 2) {
        tm.senderID = parts[2]
    }
    if (parts.length > 3) {
        tm.messageID = parts[3]
    }
    // server json-encodes data
    if (e.data) {
        tm.data = JSON.parse(e.data)
    }
    tm.messageType = e.type
    return tm
}

export async function  connectSSE(
    baseURL:string,
    ssePath:string,
    authToken:string,
    cid:string,
    handler: EventHandler,
    onProgress: (stat:DeliveryStatus)=>void,
    onConnection: ConnectionHandler ):Promise<EventSource> {

    return new Promise((resolve, reject): void => {
        // FIXME: why is no data received?
        var eventSourceInitDict = {
            headers: {
                authorization: 'bearer ' + authToken,
                origin: baseURL,
                "path": ssePath,
                "content-Type": "application/json",
                "cid": cid
            },
            https: {
                // FIXME: encrypt:false seems to be needed to be able to receive messages!???
                encrypt:false, // https://stackoverflow.com/questions/68528360/node10212dep0123deprecationwarningsetting-the-tls-servername-to-an-ipaddress
                rejectUnauthorized: false
            }
        };

        let sseURL = baseURL + ssePath
        const source = new EventSource(sseURL, eventSourceInitDict)

        source.onopen = (e: any) => {
            let cstat = ConnectionStatus.Connected
            onConnection(cstat)
            resolve(source)
        }
        source.addEventListener("ping",(e:any)=>{
            hclog.info("received ping", e)
        })

        source.addEventListener(MessageTypeProgressUpdate,(e:any)=>{
            let stat: DeliveryStatus = JSON.parse(e.data)
            onProgress(stat)
        })
        source.addEventListener(MessageTypeAction,(e:MessageEvent)=>{
           let msg = parseSSEEvent(e)
            handler(msg)
        })
        source.addEventListener(MessageTypeEvent,(e:any)=>{
            let msg = parseSSEEvent(e)
            handler(msg)
        })
        source.addEventListener(MessageTypeProperty,(e:any)=>{
            let msg = parseSSEEvent(e)
            handler(msg)
        })
        source.addEventListener("close",(e:any)=>{
            hclog.info("On close", e)
        })
        // source.addEventListener("error",(e:any)=>{
        //     hclog.error("On error", e.data)
        // })
        // source.onmessage = function (msg: any) {
        //     hclog.warn("On message", msg)
        // }
        source.onerror = function (err: any) {
            hclog.error("Connection error: " + err.message)
        }
    })
}