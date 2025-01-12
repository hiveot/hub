import {
    ConnectionHandler,
    ConnectionStatus,
    NotificationHandler, RequestHandler, ResponseHandler
} from "@hivelib/transports/IConsumerConnection.ts";

import {
    NotificationMessage,
    RequestMessage,
    ResponseMessage,
    MessageTypeNotification,
    MessageTypeResponse, MessageTypeRequest
} from "@hivelib/transports/Messages.ts";

// need access to headers for bearer token
import * as tslog from "npm:tslog";
// @ts-nocheck
import {EventSource} from "npm:launchdarkly-eventsource"
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
    onNotification: NotificationHandler,
    onRequest: RequestHandler,
    onResponse: ResponseHandler,
    onConnection: ConnectionHandler ):Promise<EventSource> {

    return new Promise((resolve, reject): void => {

        var eventSourceInitDict = {
            readTimeoutMillis: 3600000,
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

        // const httpsAgent = new https.Agent({
        //     rejectUnauthorized: false,
        // });
        // const myfetch = (init)=>
        //         fetch(sseURL, {
        //             ...init,
        //             agent: httpsAgent,
        //             headers: {
        //                 ...init.headers,
        //                 authorization: 'bearer ' + authToken,
        //                 origin: baseURL,
        //                 "path": ssePath,
        //                 "content-Type": "application/json",
        //                 "cid": cid // this header must match the ConnectionIDHeader field name on the server
        //             }
        //         })

        const source = new EventSource(sseURL,eventSourceInitDict)

        // const source = createEventSource({
        //     url: sseURL,
        //     fetch: myfetch,
        //
        //     onMessage: ({data, event, id}) => {
        //         console.log('Data: %s', data)
        //         console.log('Event ID: %s', id) // Note: can be undefined
        //         console.log('Event: %s', event) // Note: can be undefined
        //     },
        // })
        // const source = new EventSource(sseURL, {
        //     fetch: (sseURL:string,init)=>
        //         fetch(sseURL, {
        //             ...init,
        //             agent: httpsAgent,
        //             headers: {
        //                 ...init.headers,
        //                 authorization: 'bearer ' + authToken,
        //                 origin: baseURL,
        //                 "path": ssePath,
        //                 "content-Type": "application/json",
        //                 "cid": cid // this header must match the ConnectionIDHeader field name on the server
        //             }
        //         })
        //     })

            // onMessage: ({data,event,id})=>{
            //     log.info("OnMessage: event=",event)
            //
            // },
        //     onConnect: ()=>{
        //         log.info("OnConnect")
        //             let cstat = ConnectionStatus.Connected
        //             onConnection(cstat)
        //             // resolve(source)
        //     },
        //     onDisconnect: ()=>{
        //        log.info("OnDisconnect")
        //     }
        // })

        source.onopen = (e: any) => {
            let cstat = ConnectionStatus.Connected
            onConnection(cstat)
            resolve(source)
        }
        source.addEventListener("message",(e:any)=>{
            log.info("received message", e.toString())
        })
        source.addEventListener("ping",(e:any)=>{
            log.info("received ping")
        })
        source.addEventListener(MessageTypeNotification,(e:MessageEvent)=> {
            log.info("received notification", e.data)
            let req: NotificationMessage = JSON.parse(e.data as string)
            onNotification(req)
        })
        source.addEventListener(MessageTypeRequest,(e:globalThis.MessageEvent)=>{
            log.info("received request", e.data)
            let fields = JSON.parse(e.data  as string)
            let req = new RequestMessage(fields)
            onRequest(req)
        })
        source.addEventListener(MessageTypeResponse,(e:globalThis.MessageEvent)=>{
            log.info("received response", e.data)
            let req: ResponseMessage = JSON.parse(e.data  as string)
            onResponse(req)
        })
        source.addEventListener("close",(e:any)=>{
            log.info("On close")
        })
        // // source.addEventListener("error",(e:any)=>{
        // //     hclog.error("On error", e.data)
        // // })
        // // source.onmessage = function (msg: any) {
        // //     hclog.warn("On message", msg)
        // // }
        source.onerror = function (err: any) {
            if (err) {
                // TODO: differentiate between an auth error and a broken connection
                log.error("Connection error: " + baseURL + ssePath, err.message)
                // source.close()
                if (source.readyState == 2) {// EventSource.CLOSED) {
                    onConnection(ConnectionStatus.Disconnected)
                }
                reject(err)
            }
        }
    })
}