import * as tslog from 'tslog';
import {
    ConnectionHandler,
    ConnectionStatus,
    ActionHandler,
    EventHandler,
    ProgressHandler
} from "@hivelib/hubclient/IConsumerClient";

import {ThingMessage} from "@hivelib/things/ThingMessage";
import EventSource from 'eventsource'
import {ActionProgress} from "@hivelib/hubclient/ActionProgress";
import {
    MessageTypeAction,
    MessageTypeEvent,
    MessageTypeProgressUpdate,
    MessageTypeProperty
} from "@hivelib/api/vocab/vocab";

const hclog = new tslog.Logger()

export type MessageHandler = (msg:ThingMessage)=>void;


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

// Connect an EventSource to the SSE server and handle SSE events
export async function  connectSSE(
    baseURL:string,
    ssePath:string,
    authToken:string,
    cid:string,
    onMessage: MessageHandler,
    onProgress: ProgressHandler,
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
            let stat: ActionProgress = JSON.parse(e.data)
            onProgress(stat)
        })
        source.addEventListener(MessageTypeAction,(e:MessageEvent)=>{
           let msg = parseSSEEvent(e)
            onMessage(msg)
        })
        source.addEventListener(MessageTypeEvent,(e:any)=>{
            let msg = parseSSEEvent(e)
            onMessage(msg)
        })
        source.addEventListener(MessageTypeProperty,(e:any)=>{
            let msg = parseSSEEvent(e)
            onMessage(msg)
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