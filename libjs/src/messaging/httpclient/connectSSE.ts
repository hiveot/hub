import * as tslog from 'tslog';
import {
    ConnectionHandler,
    ConnectionStatus,
     RequestHandler, ResponseHandler
} from "@hivelib/messaging/IConsumerConnection";

import EventSource from 'eventsource'
import {
    RequestMessage,
    ResponseMessage,
    MessageTypeResponse, MessageTypeRequest
} from "@hivelib/messaging/Messages";

const log = new tslog.Logger({prettyLogTimeZone:"local"})


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
        // let eventSourceInit = {
        //     fetch: (input, init) =>
        //         fetch(input, {
        //             ...init,
        //             headers: {
        //                 ...init.headers,
        //                 authorization: 'bearer ' + authToken,
        //                 origin: baseURL,
        //                 "path": ssePath,
        //                 "content-Type": "application/json",
        //                 "cid": cid // this header must match the ConnectionIDHeader field name on the server
        //             },
        //             https: {
        //                 rejectUnauthorized: false
        //             }
        //         }),
        // }

        let sseURL = baseURL + ssePath
        const source = new EventSource(sseURL, eventSourceInitDict)

        source.onopen = (e: any) => {
            let cstat = ConnectionStatus.Connected
            onConnection(cstat)
            resolve(source)
        }
        source.addEventListener("ping",(e:any)=>{
            log.info("received ping", e)
        })
        source.addEventListener(MessageTypeRequest,(e:any)=>{
            log.info("received request", e)
            let fields = JSON.parse(e.data)
            let req = new RequestMessage(fields)
            onRequest(req)
        })
        source.addEventListener(MessageTypeResponse,(e:any)=>{
            log.info("received response", e)
            let req: ResponseMessage = JSON.parse(e.data)
            onResponse(req)
        })
        source.addEventListener("close",(e:any)=>{
            log.info("On close", e)
        })
        // source.addEventListener("error",(e:any)=>{
        //     hclog.error("On error", e.data)
        // })
        // source.onmessage = function (msg: any) {
        //     hclog.warn("On message", msg)
        // }
        source.onerror = function (err: any) {
            // TODO: differentiate between an auth error and a broken connection
            log.error("Connection error: "+baseURL+ssePath, err.message)
            // source.close()
            if (source.readyState == EventSource.CLOSED) {
                onConnection(ConnectionStatus.Disconnected)
            }
            reject(err)
        }
    })
}