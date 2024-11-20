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
import {ActionStatus} from "@hivelib/hubclient/ActionStatus";
import {
    OpInvokeAction,
    HTOpUpdateActionStatus,
    HTOpPublishEvent,
    HTOpUpdateProperty
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
        tm.requestID = parts[3]
    }
    // server json-encodes data
    if (e.data) {
        tm.data = JSON.parse(e.data)
    }
    tm.operation = e.type
    return tm
}

// Connect an EventSource to the SSE server and handle SSE events
// cid is the connection id field as used in all http requests. (eg, without the clientid)
export async function  connectSSE(
    baseURL:string,
    ssePath:string,
    authToken:string,
    cid:string,
    onMessage: MessageHandler,
    onProgress: ProgressHandler,
    onConnection: ConnectionHandler ):Promise<EventSource> {

    return new Promise((resolve, reject): void => {

        var eventSourceInitDict = {
            headers: {
                authorization: 'bearer ' + authToken,
                origin: baseURL,
                "path": ssePath,
                "content-Type": "application/json",
                "cid": cid // this header must match the ConnectionIDHeader field name on the server
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

        source.addEventListener(HTOpUpdateActionStatus,(e:any)=>{
            let stat: ActionStatus = JSON.parse(e.data)
            onProgress(stat)
        })
        source.addEventListener(OpInvokeAction,(e:MessageEvent)=>{
           let msg = parseSSEEvent(e)
            onMessage(msg)
        })
        source.addEventListener(HTOpPublishEvent,(e:any)=>{
            let msg = parseSSEEvent(e)
            onMessage(msg)
        })
        source.addEventListener(HTOpUpdateProperty,(e:any)=>{
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
            // TODO: differentiate between an auth error and a broken connection
            hclog.error("Connection error: " + err.message)
            // source.close()
            if (source.readyState == EventSource.CLOSED) {
                onConnection(ConnectionStatus.Disconnected)
            }
        }
    })
}