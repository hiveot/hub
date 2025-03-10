import * as tslog from 'tslog';
// undici seems to cause a warning: Cannot find module 'node:sqlite'
import { Agent, fetch } from 'undici';
import { type ConnectionHandler, ConnectionStatus, type RequestHandler, type ResponseHandler
} from "../IConsumerConnection.ts";

import {EventSource} from 'eventsource'
import {
    RequestMessage,
    ResponseMessage,
    MessageTypeResponse, MessageTypeRequest
} from "../Messages.ts";

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
    caCertPem:string,
    cid:string,
    onRequest: RequestHandler,
    onResponse: ResponseHandler,
    onConnection: ConnectionHandler ):Promise<EventSource> {

    return new Promise((resolve, reject): void => {

        // var eventSourceInitDict = {
        //     headers: {
        //         authorization: 'bearer ' + authToken,
        //         origin: baseURL,
        //         "path": ssePath,
        //         "content-Type": "application/json",
        //         "cid": cid // this header must match the ConnectionIDHeader field name on the server
        //     },
        //     https: {
        //         rejectUnauthorized: false
        //     }
        // };
        const agent = new Agent({
            connect: {
                ca: [caCertPem],
                keepAlive: true,
                allowH2: true,
                rejectUnauthorized:true,
        //         cert: readFileSync('/path/to/crt.pem', 'utf-8'),
        //         key: readFileSync('/path/to/key.pem', 'utf-8')
        }
        })
        const eventSourceInit = {
            fetch: (input:any, init:any) =>
                fetch(input, {
                    ...init,
                    headers: {
                        ...init.headers,
                        authorization: 'bearer ' + authToken,
                        origin: baseURL,
                        "path": ssePath,
                        "content-Type": "application/json",
                        "cid": cid // this header must match the ConnectionIDHeader field name on the server
                    },
                    // https: {
                    //     rejectUnauthorized: true
                    // },
                    dispatcher: agent,
                }),
        }

        const sseURL = baseURL + ssePath
        // const source = new EventSource(sseURL, eventSourceInitDict)
        const source = new EventSource(sseURL, eventSourceInit)

        source.onopen = (e: any) => {
            const cstat = ConnectionStatus.Connected
            onConnection(cstat)
            resolve(source)
        }
        source.addEventListener("ping",(e:MessageEvent)=>{
            log.info("received ping")
        })
        source.addEventListener(MessageTypeRequest,(e:MessageEvent)=>{
            log.info("received request")
            const fields = JSON.parse(e.data)
            const req = new RequestMessage(fields)
            onRequest(req)
        })
        source.addEventListener(MessageTypeResponse,(e:MessageEvent)=>{
            log.info("received response")
            const req: ResponseMessage = JSON.parse(e.data)
            onResponse(req)
        })
        source.addEventListener("close",(e:MessageEvent)=>{
            log.info("On close")
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