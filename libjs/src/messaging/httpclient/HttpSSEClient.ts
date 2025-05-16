import * as tslog from 'tslog';
import * as http2 from "node:http2";
import {nanoid} from "nanoid";
import {Buffer} from "node:buffer";

import TD, {TDForm} from '../../wot/TD.ts';
import type IAgentConnection from "../IAgentConnection.ts";
import {
    OpInvokeAction,
    OpWriteProperty,
    OpSubscribeEvent,
    OpSubscribeAllEvents,
    OpUnsubscribeAllEvents,
    OpUnsubscribeEvent,
    OpObserveAllProperties,
    OpObserveProperty,
} from "../../api/vocab/vocab.js";
import {connectSSE} from "./connectSSE.ts";
import {
    type RequestHandler,
    type ResponseHandler, ConnectionStatus
} from "../IConsumerConnection.ts";
import {NotificationMessage, RequestMessage, ResponseMessage} from "../Messages.ts";
import {EventSource} from "eventsource";

// FIXME: import from vocab is not working
const RequestCompleted = "completed"
const RequestFailed = "failed"


// HTTP protocol and subprotocol constants
// These should eventually be phased out in favor of forms, where possible.

// Keep this in sync with transports/servers/httpserver/HttpRouter.go
//
// FIXME: THESE WILL BE REMOVED WHEN SWITCHING TO FORMS

// TODO: use directory service
const ThingDirectoryDThingID = "dtw:digitwin:ThingDirectory"
const ThingDirectoryUpdateTDMethod = "updateTD"


// HTTP Paths for auth.
// THIS WILL BE REMOVED AFTER THE PROTOCOL BINDING PUBLISHES THESE IN THE TDD.
// The hub client will need the TD (ConsumedThing) to determine the paths.
const HttpPostLoginPath   = "/authn/login"
const HttpPostLogoutPath  = "/authn/logout"
const HttpPostRefreshPath = "/authn/refresh"
const HttpGetDigitwinPath = "/digitwin/{operation}/{thingID}/{name}"

// Generic form href that maps to all operations for the http client, using URI variables
// Generic HiveOT HTTP urls when Forms are not available. The payload is a
// corresponding standardized message.
const HiveOTPostRequestHRef      = "/hiveot/request"
const HiveOTPostResponseHRef     = "/hiveot/response"
const HiveOTPostNotificationHRef = "/hiveot/notification"

const hcLog = new tslog.Logger({prettyLogTimeZone:"local"})

// HttpSSEClient implements the javascript client for connecting to the hub
// using HiveOT's HTTPS and SSE-SC protocol for the return channel.
export default class HttpSSEClient implements IAgentConnection {
    _clientID: string;
    _baseURL: string;
    _caCertPem: string;
    _disableCertCheck: boolean
    _http2Session: http2.ClientHttp2Session | undefined;
    _ssePath: string;
    _sseClient: EventSource | undefined;
    _cid:string;

    isInitialized: boolean = false;
    sseConnStatus: ConnectionStatus;
    // the auth token when using connectWithToken
    authToken: string;

    // client handler for connection status change
    connectHandler?: (status: ConnectionStatus) => void;
    // request handler (agents only) for received action and property write requests.
    requestHandler?: RequestHandler;
    // response handler for receiving responses to non-rpc requests
    responseHandler?: ResponseHandler;
    // this client does not receive notifications

    // the provider of forms for an operation
    getForm?: (op: string) => TDForm;

    // map of correlationID to response handlers update channel
    _correlData: Map<string,(resp: ResponseMessage)=>void>

    // Instantiate the Hub Client.
    //
    // The flag disableCertCheck is intended for use with self-signed certificate
    // on the local network where the CA is not available.
    //
    // @param fullURL: of HTTP/SSE server to connect to or empty for auto-discovery
    // @param clientID: connected as this client
    // @param caCertPem: verify server against this CA certificate
    // @param disableCertCheck: don't check the (self signed) server certificate
    constructor(fullURL: string, clientID: string, caCertPem: string, disableCertCheck: boolean) {
        // this._baseURL = hostPort;
        this._caCertPem = caCertPem;
        this._clientID = clientID;
        this._disableCertCheck = disableCertCheck;
        // this._ssePath = DefaultSSESCPath;
        this.sseConnStatus = ConnectionStatus.Disconnected;
        this.authToken = "";
        this._correlData = new Map();
        this._cid = nanoid() // connection id

        const url = new URL(fullURL)
        this._baseURL = "https://"+url.host
        this._ssePath = url.pathname
    }

    // ClientID the client is authenticated as to the server
    get clientID(): string {
        return this._clientID;
    }

    // return the current connection status
    get connectionStatus(): { status: ConnectionStatus } {
        return {status: this.sseConnStatus};
    }

    // setup a TLS connection with the hub.
    // This just creates a http2 connection. It does not establish an SSE, nor authenticates.
    async connect(): Promise<http2.ClientHttp2Session> {

        if (this._disableCertCheck) {
            hcLog.warn("Disabling server certificate check.")
        }
        const opts: http2.SecureClientSessionOptions = {
            timeout: 10000, // msec???
            "rejectUnauthorized": !this._disableCertCheck
        }
        if (this._caCertPem) {
            opts.ca = this._caCertPem
        }
        // http connection
        this._http2Session = http2.connect(this._baseURL, opts)

        // When an error occurs, show it.
        this._http2Session.on('close', () => {
            console.warn("http/2 connection has closed");
            // don't do anything here. on the next post-message a reconnect will be attempted.
            // the sse connection is more important
        });
        this._http2Session.on('connect', (ev) => {
            console.warn("http/2 connected to server, cid=",this._cid,"url",this._baseURL);
            // the sse connection is more important
        });
        this._http2Session.on('error', (error) => {
            console.error("http/2 connection error: "+error);
            // the sse connection is more important
        });
        this._http2Session.on('frameError', (error) => {
            console.error(error);
        });
        this._http2Session.on('end', () => {
            // TODO: determine when this gets called
            console.log('http/2 server ended the connection?');
            this.disconnect()
        });

        return this._http2Session
    }

    // ConnectWithPassword connects to the Hub server using the clientID and password.
    async connectWithPassword(password: string): Promise<string> {
        // establish an http/2 connection instance
        await this.connect()
        // invoke a login request
        const loginArgs = {
            "login": this._clientID,
            "password": password,
        }
        const resp = await this.pubMessage("POST", HttpPostLoginPath,loginArgs,"")

        const loginResp = JSON.parse(resp)
        this.authToken = loginResp.token
        // with the new auth token a SSE return channel can be established
        this._sseClient = await connectSSE(
            this._baseURL, this._ssePath, this.authToken, this._caCertPem, this._cid,
            this.onRequest.bind(this),
            this.onResponse.bind(this),
            this.onSseConnection.bind(this))

        return loginResp.token
    }

    // connect and login to the Hub gateway using a JWT token
    // host is the server address
    async connectWithToken(authToken: string): Promise<string> {
        this.authToken = authToken
        await this.connect()
        this._sseClient = await connectSSE(
            this._baseURL, this._ssePath, this.authToken, this._caCertPem, this._cid,
            this.onRequest.bind(this),
            this.onResponse.bind(this),
            this.onSseConnection.bind(this))
        return ""
    }


    // disconnect if connected
    async disconnect() {
        if (this._sseClient && this._sseClient.close) {
            this._sseClient.close()
            this._sseClient = undefined
        }
        if (this._http2Session) {
            this._http2Session.close();
            this._http2Session = undefined
        }
        if (this.sseConnStatus != ConnectionStatus.Disconnected) {
            this.sseConnStatus = ConnectionStatus.Disconnected
        }
    }

    // TODO: logout

    // callback handler invoked when the SSE connection status has changed
    onSseConnection(status: ConnectionStatus) {
        this.sseConnStatus = status
        if (this.sseConnStatus === ConnectionStatus.Connected) {
            hcLog.info('HubClient SSE connected to '+ this._baseURL+this._ssePath + ' as '+this._clientID);
        } else if (this.sseConnStatus == ConnectionStatus.Connecting) {
            hcLog.warn('HubClient SSE attempt connecting');
        } else {
            hcLog.warn('HubClient SSE disconnected. Will attempt to reconnect');
            // TODO: can this be done here in the callback?
            this._sseClient = undefined
            this.connectWithToken(this.authToken).then()
        }
    }

    // Handle incoming request (as an agent) and pass them to the registered handler
    onRequest(req: RequestMessage):  ResponseMessage|null {
        let resp: ResponseMessage|null
        try {
            if (this.requestHandler) {
                resp = this.requestHandler(req)
            } else {
                const err = Error(`onRequest: received request but no handler registered: ${req.operation}`)
                hcLog.warn(err)
                resp = req.createResponse(null, err)
            }
        } catch (e) {
            const err = Error(`Error handling request sender=${req.senderID}, messageType=${req.operation}, thingID=${req.thingID}, name=${req.name}, error=${e}`)
            hcLog.warn(err)
            resp = req.createResponse(null,err)
        }
        // only send a response if one is available
        if (resp) {
            this.sendResponse(resp)
        }
        return resp
    }

    // Handle response to previous sent request
    onResponse(resp:ResponseMessage):void{
        let cb: ResponseHandler|undefined

        if (resp.correlationID) {
            cb = this._correlData.get(resp.correlationID)
        }
        if (cb) {
            cb(resp)
        } else if (this.responseHandler ) {
            this.responseHandler(resp)
        }
    }

    // try using pubMessage using fetch api
    // HOW TO SET THE CA? https://sebtrif.xyz/blog/2019-10-03-client-side-ssl-in-node-js-with-fetch/
    // async pubMessage(path:string, payload: string):Promise<string> {
    //     return new Promise((resolve,reject)=> {
    //
    //         const req = https.request({
    //             hostname: this._baseURL,
    //             origin: this._baseURL,
    //             auth:  "bearer "+this.authToken,
    //             path: path,
    //             method: "POST",
    //             ca: this._caCertPem,
    //             "content-type": "application/json",
    //             // "content-length": Buffer.byteLength(payload),
    //
    //         },
    //         res => {
    //             res.on('data',function(data){
    //
    //              });
    //         })
    //         req.headers[""] = ""
    //     })
    // }

    // publish a http request to the path
    // if the http/2 connection is closed, then try to initialize it again.
    async pubMessage(methodName: string, path: string, data: any, correlationID?:string):Promise<string> {
        // if the session is invalid, restart it
        if (!this._http2Session || this._http2Session.destroyed) {
             hcLog.warn("http2.pubMessage but connection is closed. Attempting to reconnect")
            await this.connect()
        }

        return  new Promise((resolve, reject) => {
            let replyData: string = ""
            let statusCode: number
            const payload = JSON.stringify(data)

            if (!this._http2Session || this._http2Session.closed) {
                // getting here is weird. this._http2Session is undefined while
                // the debugger shows a value.
                reject(new Error("pubMessage: Unable to send. http2 connection was closed"))
            } else {
                const h2req = this._http2Session.request({
                    origin: this._baseURL,
                    authorization: "bearer " + this.authToken,
                    ':path': path,
                    ":method": methodName,
                    "content-type": "application/json",
                    "content-length": Buffer.byteLength(payload),
                    "message-id": correlationID,
                    "cid": this._cid,
                })

                h2req.setEncoding('utf8');

                h2req.on('response', (r) => {
                    if (r[":status"]) {
                        statusCode = r[":status"]
                        if (statusCode >= 400) {
                            hcLog.warn(`pubMessage '${path}' returned status code '${statusCode}'`)
                        }
                    }
                })
                h2req.on('data', (chunk) => {
                    replyData = replyData + chunk
                });
                h2req.on('end', () => {
                    h2req.destroy()
                    if (statusCode >= 400) {
                        hcLog.warn(`pubMessage http/2 connection closed with status code  ${statusCode}`)
                        reject(new Error("Error " + statusCode + ": " + replyData))
                    } else {
                        // hclog.info(`pubMessage to ${path}. Received reply. size=` + replyData.length)
                        // reject(new Error(replyData))
                        resolve(replyData)
                    }
                });
                h2req.on('error', (err) => {
                    hcLog.warn(`pubMessage http/2 error: ${err}`)
                    h2req.destroy()
                    reject(err)
                });
                // write the body and complete the request
                h2req.end(payload)
            }
        })
    }

    // invokeAction publishes a request for action from a Thing.
    // This is a simple helper that uses sendRequest(...)
    //
    //	@param agentID: of the device or service that handles the action.
    //	@param thingID: is the destination thingID to whom the action applies.
    //	name is the name of the action as described in the Thing's TD
    //	input is the optional action input to be serialized and transported
    //
    // This returns the response message
    async invokeAction(thingID: string, name: string, input: any): Promise<ResponseMessage> {

        hcLog.info("pubAction. thingID:", thingID, ", name:", name)
        const req = new RequestMessage({
            operation:OpInvokeAction,
            thingID:thingID,
            name:name,
            input:input
        })
        return this.sendRequest(req)
        //
        // let actionPath = PostInvokeActionPath.replace("{thingID}", thingID)
        // actionPath = actionPath.replace("{name}", name)
        //
        // let resp = await this.pubMessage("POST",actionPath, correlationID, payload)
        // let stat: ActionStatus = JSON.parse(resp)
        // return stat
    }


    // PubEvent - Agent publishes a Thing event for event subscribers.
    // The payload is an event value as per event affordance.
    // Intended for agents of devices and services to notify of changes to the Things
    // they are the agent for.
    //
    // thingID is the ID of the 'thing' whose event to publish.
    // This is the ID under which the TD document is published that describes
    // the thing. It can be the ID of the sensor, actuator or service.
    //
    // This sends a response message with the SubscribeEvent operation with status
    // completed.
    //
    //	@param thingID: of the Thing whose event is published
    //	@param eventName: is one of the predefined events as described in the Thing TD
    //	@param payload: is the serialized event value, or nil if the event has no value
    pubEvent(thingID: string, name: string, data: any) {

        hcLog.info("pubEvent. thingID:", thingID, ", name:", name)
        const msg = new NotificationMessage(OpSubscribeEvent, thingID,name,data)
        this.sendNotification(msg)

        // let eventPath = PostAgentPublishEventPath.replace("{thingID}", thingID)
        // eventPath = eventPath.replace("{name}", name)
        //
        // this.pubMessage("POST",eventPath, "", payload)
        //     .catch((e)=> {
        //         hclog.warn("failed publishing event", e)
        //     }
        // )
    }


    // Publish batch of property values to property observers
    pubMultipleProperties(thingID: string, propMap: { [key: string]: any }) {

        hcLog.info("pubMultipleProperties. thingID:", thingID)
        const msg = new NotificationMessage(OpObserveAllProperties, thingID,"",propMap)
        this.sendNotification(msg)
    }

    // Publish thing property value update to property observers
    pubProperty(thingID: string, name:string, value: any) {
        hcLog.info("pubProperty. thingID:", thingID," name:",name,"value",value)
        const msg = new NotificationMessage(OpObserveProperty, thingID,name,value)
        this.sendNotification(msg)
    }

    // PubTD publishes a req with a Thing TD document.
    // This serializes the TD into JSON as per WoT specification
    pubTD(td: TD) {
        hcLog.info("pubTD. thingID:", td.id)
        const tdJSON = JSON.stringify(td, null, ' ');

        // Invoke action to update the directory service
        // TODO: convert to use the discovered directory
        this.invokeAction(ThingDirectoryDThingID, ThingDirectoryUpdateTDMethod,tdJSON)
            .then((resp: ResponseMessage) => {
                // complete the request if the result is returned, otherwise wait for
                // the callback from _correlData
                console.info("TD published: "+td.id)
            })
            .catch((e) => {
                console.error("pubTD failed", e);
            })
        // let msg = new RequestMessage({
        //     operation:HTOpUpdateTD,
        //     thingID: td.id,
        //     input: tdJSON,
        // })
        // return this.sendRequest(msg)
    }

    // obtain a new token
    async refreshToken(): Promise<string> {

        let refreshPath = HttpPostRefreshPath.replace("{thingID}", "authn")
        refreshPath = refreshPath.replace("{name}", "refreshMethod")
        // TODO use generated API
        const args = {
            clientID: this.clientID,
            oldToken: this.authToken,
        }
        try {
            const resp = await this.pubMessage("POST",refreshPath, args,"");
            this.authToken = JSON.parse(resp)
            return this.authToken
        } catch (e) {
            hcLog.error("refreshToken failed: ", e)
            throw e
        }
    }
    //
    // // sendNotification [agent] sends a notification message to the hub.
    // // @param msg: notification to send
    // sendNotification(notif: ResponseMessage) {
    //     // notifications can only be send to the fixed endpoint
    //     this.pubMessage(
    //         "POST",HiveOTPostNotificationHRef, notif)
    //         .then().catch()
    // }

    // sendResponse [agent] sends a response status message to the hub.
    // @param resp: response to send
   async sendRequest(req: RequestMessage):Promise<ResponseMessage> {
        return new Promise((resolve, _reject): void => {
            // use forms for requests for interoperability
            let href: string | undefined
            let f: TDForm | undefined
            if (this.getForm) {
                f = this.getForm(req.operation)
            }
            if (f) {
                href = f.getHRef()
            }
            if (href) {
                // using a form means following http-basic
                this.pubMessage("POST", href, req.input, req.correlationID)
                    .then((reply: string) => {
                        const value = JSON.parse(reply)
                        const resp = new ResponseMessage(
                            req.operation, req.thingID, req.name, value, "", "")
                        resolve(resp)
                    })
            } else {
                // no form, use the 'well-known' RequestMessage path
                this.pubMessage("POST", HiveOTPostRequestHRef, req, req.correlationID)
                    .then((reply: string) => {
                        let resp: ResponseMessage;
                        if (reply) {
                            resp = JSON.parse(reply)
                        } else {
                            const err = new Error("Response without responseMessage envelope")
                            resp = req.createResponse(undefined, err)
                        }
                        resolve(resp)
                    })
            }
        })
    }

    // sendNotification [agent] sends a notification message to the hub.
    // @param notif: notification to send
    sendNotification(notif: NotificationMessage) {
        // responses can only be sent to the fixed endpoint
        this.pubMessage(
            "POST",HiveOTPostNotificationHRef, notif,notif.correlationID)
            .then()
            .catch((e) => {
                console.log("sendNotification error", e);
            })
    }

    // sendResponse [agent] sends a response status message to the hub.
    // @param resp: response to send
    sendResponse(resp: ResponseMessage) {
        // responses can only be sent to the fixed endpoint
        this.pubMessage(
            "POST",HiveOTPostResponseHRef, resp,resp.correlationID)
            .then()
            .catch((e) => {
                console.log("sendResponse error: ", e);
            })
    }

    // Rpc publishes an RPC request to a service and waits for a response.
    // Intended for users and services to invoke RPC to services.
    // This return the response data.
    async rpc(dThingID: string, methodName: string, args: any): Promise<any> {
        return new Promise((resolve, reject) => {

            // a correlationID is needed before the action is published in order to match it with the reply
            const correlationID = "rpc-" + nanoid()

            // handle timeout
            const t1 = setTimeout(() => {
                this._correlData.delete(correlationID)
                console.error("RPC",dThingID,methodName,"failed with timeout")
                reject("timeout")
            }, 30000)

            // set the handler for response messages
            this._correlData.set(correlationID, (resp:ResponseMessage):void=> {
                // console.log("delivery progress",stat.progress)
                // Remove the rpc wait hook and resolve the rpc
                clearTimeout(t1)
                this._correlData.delete(correlationID)
                resolve(resp)
            })
            this.invokeAction(dThingID, methodName, args)
                .then((resp: ResponseMessage) => {
                    // complete the request if the result is returned, otherwise wait for
                    // the callback from _correlData
                    resolve(resp.value)
                })
                .catch((e) => {
                    console.error("RPC failed", e);
                    reject(e)
                })
        })
    }


    // Read Thing definitions from the directory
    // @param publisherID whose things to read or "" for all publishers
    // @param thingID whose to read or "" for all things of the publisher(s)
    // async readDirectory(agentID: string, thingID: string): Promise<string> {
    // 	return global.hapiReadDirectory(publisherID, thingID);
    // }

    // application connect/disconnect handler
    setConnectHandler(handler: (status: ConnectionStatus) => void): void {
        this.connectHandler = handler
    }
    // set the handler of incoming requests such as action or property write requests
    //
    // The handler should return a ResponseMessage containing the handling status.
    setRequestHandler(handler: RequestHandler) {
        this.requestHandler = handler
    }
    setResponseHandler(handler: ResponseHandler) {
        this.responseHandler = handler
    }

    // Subscribe to events from other things.
    //
    // The events will be passed to the configured onEvent handler.
    //
    // note there is no unsubscribe. The intended use is to subscribe to devices/things/events
    // of interest and leave it at that. Currently there is no use-case that requires
    // a frequent subscribe/unsubscribe.
    //
    // @param dThingID: optional filter of the thing whose events are published; "" for all things
    // @param name: optional filter on the event name; "" for all event names.
    subscribe(dThingID: string, name: string):void {
        let op = OpSubscribeEvent
        if (!name) {
            op = OpSubscribeAllEvents
        }
        const req = new RequestMessage({
            operation:op, thingID:dThingID, name:name
        })
        this.sendRequest(req)
            .then().catch()
    }

    unsubscribe(dThingID: string, name: string) {
        let op = OpUnsubscribeEvent
        if (!name) {
            op = OpUnsubscribeAllEvents
        }
        const req = new RequestMessage({
            operation:op, thingID:dThingID, name:name
        })
        this.sendRequest(req)
            .then().catch()
    }

    // writeProperty publishes a request for changing a Thing's property.
    // The configuration is a writable property as defined in the Thing's TD.
    writeProperty(thingID: string, name: string, propValue: any) {
        hcLog.info("writeProperty. thingID:", thingID, ", name:", name)

        const req = new RequestMessage({
            operation: OpWriteProperty, thingID: thingID, name: name, input: propValue
        })
        return this.sendRequest(req)
    }
}
