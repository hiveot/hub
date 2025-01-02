import * as tslog from 'tslog';
import {
    ConnectionHandler,
    ConnectionStatus,
    NotificationHandler, RequestHandler, ResponseHandler
} from "@hivelib/transports/IConsumerConnection";

import EventSource from 'eventsource'
import {
    OpInvokeAction,
    HTOpPublishEvent,
    HTOpUpdateProperty
} from "@hivelib/api/vocab/vocab";
import {NotificationMessage, RequestMessage, ResponseMessage, MessageTypeNotification} from "@hivelib/transports/Messages";

const hclog = new tslog.Logger()

// function parseSSERequest(e: MessageEvent) : RequestMessage {
//     let tm = new RequestMessage()
//     if (e.data) {
//         tm = JSON.parse(e.data)
//     }
//     return tm
// }

// Connect an EventSource to the SSE server and handle SSE events
// cid is the connection id field as used in all http requests. (eg, without the clientid)
export async function  connectSSE(
    baseURL:string,
    ssePath:string,
    authToken:string,
    cid:string,
    onNotification: NotificationHandler,
    onRequest: RequestHandler,
    onResponse: ResponseHandler,
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
        source.addEventListener(MessageTypeNotification,(e:any)=>{
            hclog.info("received notification", e)
            let req: NotificationMessage = JSON.parse(e.data)
            onNotification(req)
        })
        source.addEventListener("request",(e:any)=>{
            hclog.info("received request", e)
            let req: RequestMessage = JSON.parse(e.data)
            onRequest(req)
        })
        source.addEventListener("response",(e:any)=>{
            hclog.info("received response", e)
            let req: ResponseMessage = JSON.parse(e.data)
            onResponse(req)
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