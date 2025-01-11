import {TD, TDForm} from '../../wot/TD.js';
import {
    IAgentConnection,
} from "../IAgentConnection.js";
import * as tslog from 'tslog';
import {
    OpInvokeAction,
    HTOpPublishEvent,
    HTOpUpdateProperty,
    OpWriteProperty,
    HTOpUpdateMultipleProperties,
    OpSubscribeEvent,
    OpSubscribeAllEvents, OpUnsubscribeAllEvents, OpUnsubscribeEvent, HTOpUpdateTD,
} from "@hivelib/api/vocab/vocab.js";
import * as http2 from "node:http2";
import {connectSSE} from "@hivelib/transports/httpclient/connectSSE";
import {
    RequestHandler,
    NotificationHandler,
    ResponseHandler, ConnectionStatus
} from "@hivelib/transports/IConsumerConnection";
import {nanoid} from "nanoid";
import EventSource from "eventsource";
import {NotificationMessage, RequestMessage, ResponseMessage} from "@hivelib/transports/Messages";

// FIXME: import from vocab is not working
const RequestCompleted = "completed"
const RequestFailed = "failed"


// HTTP protocol and subprotocol constants
// These should eventually be phased out in favor of forms, where possible.

// Keep this in sync with transports/servers/httpserver/HttpRouter.go
//
// FIXME: THESE WILL BE REMOVED WHEN SWITCHING TO FORMS



// HTTP protoocol constants
// StatusHeader contains the result of the request, eg Pending, Completed or Failed
const StatusHeader = "status"
// CorrelationIDHeader for transports that support headers can include a message-ID
const CorrelationIDHeader = "correlation-id"
// ConnectionIDHeader identifies the client's connection in case of multiple
// connections from the same client.
const ConnectionIDHeader = "cid"
// DataSchemaHeader to indicate which  'additionalresults' dataschema being returned.
const DataSchemaHeader = "dataschema"


// HTTP Paths for auth.
// THIS WILL BE REMOVED AFTER THE PROTOCOL BINDING PUBLISHES THESE IN THE TDD.
// The hub client will need the TD (ConsumedThing) to determine the paths.
const HttpPostLoginPath   = "/authn/login"
const HttpPostLogoutPath  = "/authn/logout"
const HttpPostRefreshPath = "/authn/refresh"
const HttpGetDigitwinPath = "/digitwin/{operation}/{thingID}/{name}"

// paths for HTTP subprotocols
const DefaultWSSPath   = "/wss"
const DefaultSSEPath   = "/sse"
const DefaultSSESCPath = "/ssesc"

// Generic form href that maps to all operations for the http client, using URI variables
// Generic HiveOT HTTP urls when Forms are not available. The payload is a
// corresponding standardized message.
const HiveOTPostNotificationHRef = "/hiveot/notification"
const HiveOTPostRequestHRef      = "/hiveot/request"
const HiveOTPostResponseHRef     = "/hiveot/response"



const hclog = new tslog.Logger({prettyLogTimeZone:"local"})

// HubClient implements the javascript client for connecting to the hub
// using HTTPS and SSESC for the return channel.
export class HttpSSEClient implements IAgentConnection {
    _clientID: string;
    _baseURL: string;
    _caCertPem: string;
    _disableCertCheck: boolean
    _http2Session: http2.ClientHttp2Session | undefined;
    _ssePath: string;
    _sseClient: any;
    _cid:string;

    isInitialized: boolean = false;
    connStatus: ConnectionStatus;
    // the auth token when using connectWithToken
    authToken: string;

    // client handler for connection status change
    connectHandler?: (status: ConnectionStatus) => void;
    // notification handler for handling event and property update notifications
    notificationHandler?: NotificationHandler;
    // request handler (agents only) for received action and property write requests.
    requestHandler?: RequestHandler;
    // response handler for receiving responses to non-rpc requests
    responseHandler?: ResponseHandler;

    // the provider of forms for an operation
    getForm?: (op: string) => TDForm;

    // map of correlationID to response handlers update channel
    _correlData: Map<string,(resp: ResponseMessage)=>void>

    // Instantiate the Hub Client.
    //
    // The flag disableCertCheck is intended for use with self-signed certificate
    // on the local network where the CA is not available.
    //
    // @param fullURL: of HTTP/SSESC server to connect to or empty for auto-discovery
    // @param clientID: connected as this client
    // @param caCertPem: verify server against this CA certificate
    // @param disableCertCheck: don't check the (self signed) server certificate
    constructor(fullURL: string, clientID: string, caCertPem: string, disableCertCheck: boolean) {
        // this._baseURL = hostPort;
        this._caCertPem = caCertPem;
        this._clientID = clientID;
        this._disableCertCheck = disableCertCheck;
        // this._ssePath = DefaultSSESCPath;
        this.connStatus = ConnectionStatus.Disconnected;
        this.authToken = "";
        this._correlData = new Map();
        this._cid = nanoid() // connection id

        let url = new URL(fullURL)
        this._baseURL = url.origin
        this._ssePath = url.pathname
    }

    // ClientID the client is authenticated as to the server
    get clientID(): string {
        return this._clientID;
    }

    // return the current connection status
    get connectionStatus(): { status: ConnectionStatus } {
        return {status: this.connStatus};
    }

    // setup a TLS connection with the hub.
    // This just creates a http2 connection. It does not establish an SSE, nor authenticates.
    async connect(): Promise<http2.ClientHttp2Session> {

        if (this._disableCertCheck) {
            hclog.warn("Disabling server certificate check.")
        }
        let opts: http2.SecureClientSessionOptions = {
            timeout: 10000, // msec???
            "rejectUnauthorized": !this._disableCertCheck
        }
        if (!!this._caCertPem) {
            opts.ca = this._caCertPem
        }
        // http connection
        this._http2Session = http2.connect(this._baseURL, opts)

        // When an error occurs, show it.
        this._http2Session.on('close', () => {
            console.warn("connection has closed");
            // don't do anything here. on the next post-message a reconnect will be attempted.
            // this.disconnect()
        });
        this._http2Session.on('connect', (ev) => {
            console.warn("connected to server, cid=",this._cid);
        });
        this._http2Session.on('error', (error) => {
            console.error("connection error: "+error);
        });
        this._http2Session.on('frameError', (error) => {
            console.error(error);
        });
        this._http2Session.on('end', () => {
            // TODO: determine when this gets called
            console.log('server ended the connection?');
            this.disconnect()
        });

        return this._http2Session
    }

    // ConnectWithPassword connects to the Hub server using the clientID and password.
    async connectWithPassword(password: string): Promise<string> {
        // establish an http/2 connection instance
        await this.connect()
        // invoke a login request
        let loginArgs = {
            "login": this._clientID,
            "password": password,
        }
        let resp = await this.pubMessage("POST", HttpPostLoginPath,loginArgs,"")

        let loginResp = JSON.parse(resp)
        this.authToken = loginResp.token
        // with the new auth token a SSE return channel can be established
        this._sseClient = await connectSSE(
            this._baseURL, this._ssePath, this.authToken, this._cid,
            this.onNotification.bind(this),
            this.onRequest.bind(this),
            this.onResponse.bind(this),
            this.onConnection.bind(this))

        return loginResp.token
    }

    // connect and login to the Hub gateway using a JWT token
    // host is the server address
    async connectWithToken(authToken: string): Promise<string> {
        this.authToken = authToken
        await this.connect()
        this._sseClient = await connectSSE(
            this._baseURL, this._ssePath, this.authToken, this._cid,
            this.onNotification.bind(this),
            this.onRequest.bind(this),
            this.onResponse.bind(this),
            this.onConnection.bind(this))
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
        if (this.connStatus != ConnectionStatus.Disconnected) {
            this.connStatus = ConnectionStatus.Disconnected
        }
    }

    // TODO: logout

    // callback handler invoked when the SSE connection status has changed
    onConnection(status: ConnectionStatus) {
        this.connStatus = status
        if (this.connStatus === ConnectionStatus.Connected) {
            hclog.info('HubClient connected to '+ this._baseURL+this._ssePath + ' as '+this._clientID);
        } else if (this.connStatus == ConnectionStatus.Connecting) {
            hclog.warn('HubClient attempt connecting');
        } else {
            hclog.warn('HubClient disconnected');
            // todo: retry connecting
        }
    }

    // Handle incoming event or property notifications and pass them to the handler
    onNotification(msg: NotificationMessage): void {
        try {
            if (this.notificationHandler) {
                this.notificationHandler(msg)
            } else {
                hclog.warn(`onNotification: received notification but no handler registered: ${msg.operation}`)
            }
        } catch (e) {
            let errText = `Error handling hub notification sender=${msg.senderID}, messageType=${msg.operation}, thingID=${msg.thingID}, name=${msg.name}, error=${e}`
            hclog.warn(errText)
        }
    }

    // Handle incoming request (as an agent) and pass them to the registered handler
    onRequest(req: RequestMessage):  ResponseMessage {
        let resp: ResponseMessage
        try {
            if (this.requestHandler) {
                resp = this.requestHandler(req)
            } else {
                let err = Error(`onRequest: received request but no handler registered: ${req.operation}`)
                hclog.warn(err)
                resp = req.createResponse(null, err)
            }
        } catch (e) {
            let err = Error(`Error handling request sender=${req.senderID}, messageType=${req.operation}, thingID=${req.thingID}, name=${req.name}, error=${e}`)
            hclog.warn(err)
            resp = req.createResponse(null,err)
            resp.received = req.created
        }
        this.sendResponse(resp)
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

    // publish a request to the path with the given data
    // if the http/2 connection is closed, then try to initialize it again.
    async pubMessage(methodName: string, path: string, data: any, correlationID?:string):Promise<string> {
        // if the session is invalid, restart it
        if (!this._http2Session || this._http2Session.closed) {
            // this._http2Client.
            hclog.error("pubMessage but connection is closed")
            await this.connect()
        }

        return  new Promise((resolve, reject) => {
            let replyData: string = ""
            let statusCode: number
            let payload = JSON.stringify(data)

            if (!this._http2Session || this._http2Session.closed) {
                // getting here is weird. this._http2Session is undefined while
                // the debugger shows a value.
                reject(new Error("Unable to send. Connection was closed"))
            } else {
                let req = this._http2Session.request({
                    origin: this._baseURL,
                    authorization: "bearer " + this.authToken,
                    ':path': path,
                    ":method": methodName,
                    "content-type": "application/json",
                    "content-length": Buffer.byteLength(payload),
                    "message-id": correlationID,
                    "cid": this._cid,
                })

                req.setEncoding('utf8');

                req.on('response', (r) => {
                    if (r[":status"]) {
                        statusCode = r[":status"]
                        if (statusCode >= 400) {
                            hclog.warn(`pubMessage '${path}' returned status code '${statusCode}'`)
                        }
                    }
                })
                req.on('data', (chunk) => {
                    replyData = replyData + chunk
                });
                req.on('end', () => {
                    req.destroy()
                    if (statusCode >= 400) {
                        hclog.warn(`pubMessage status code  ${statusCode}`)
                        reject(new Error("Error " + statusCode + ": " + replyData))
                    } else {
                        // hclog.info(`pubMessage to ${path}. Received reply. size=` + replyData.length)
                        reject(new Error(replyData))
                    }
                });
                req.on('error', (err) => {
                    req.destroy()
                    reject(err)
                });
                // write the body and complete the request
                req.end(payload)

                resolve(replyData)
            }
        })
    }

    // invokeAction publishes a request for action from a Thing.
    // This is a simple helper that uses sendRequest(wot.OpInvokeAction, ...)
    //
    //	@param agentID: of the device or service that handles the action.
    //	@param thingID: is the destination thingID to whom the action applies.
    //	name is the name of the action as described in the Thing's TD
    //	input is the optional action input to be serialized and transported
    //
    // This returns the response message
    async invokeAction(thingID: string, name: string, input: any): Promise<ResponseMessage> {

        hclog.info("pubAction. thingID:", thingID, ", name:", name)
        let req = new RequestMessage({
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


    // PubEvent - Agent publishes a Thing event.
    // The payload is an event value as per event affordance.
    // Intended for agents of devices and services to notify of changes to the Things
    // they are the agent for.
    //
    // thingID is the ID of the 'thing' whose event to publish.
    // This is the ID under which the TD document is published that describes
    // the thing. It can be the ID of the sensor, actuator or service.
    //
    // This will use the client's ID as the agentID of the event.
    // eventName is the ID of the event described in the TD document 'events' section,
    // or one of the predefined events listed above as EventIDXyz
    //
    //	@param thingID: of the Thing whose event is published
    //	@param eventName: is one of the predefined events as described in the Thing TD
    //	@param payload: is the serialized event value, or nil if the event has no value
    pubEvent(thingID: string, name: string, data: any) {

        hclog.info("pubEvent. thingID:", thingID, ", name:", name)
        let msg = new NotificationMessage(HTOpPublishEvent, thingID,name,data)
        return this.sendNotification(msg)

        // let eventPath = PostAgentPublishEventPath.replace("{thingID}", thingID)
        // eventPath = eventPath.replace("{name}", name)
        //
        // this.pubMessage("POST",eventPath, "", payload)
        //     .catch((e)=> {
        //         hclog.warn("failed publishing event", e)
        //     }
        // )
    }


    // Publish batch of property values
    pubMultipleProperties(thingID: string, propMap: { [key: string]: any }) {

        hclog.info("pubMultipleProperties. thingID:", thingID)
        let msg = new NotificationMessage(HTOpUpdateMultipleProperties, thingID,"",propMap)
        return this.sendNotification(msg)
    }

    // Publish thing property value update
    pubProperty(thingID: string, name:string, value: any) {
        hclog.info("pubProperty. thingID:", thingID)
        let msg = new NotificationMessage(HTOpUpdateProperty, thingID,name,value)
        return this.sendNotification(msg)
    }

    // PubTD publishes an event with a Thing TD document.
    // This serializes the TD into JSON as per WoT specification
    pubTD(td: TD) {
        hclog.info("pubTD. thingID:", td.id)
        let tdJSON = JSON.stringify(td, null, ' ');
        let msg = new NotificationMessage(HTOpUpdateTD, td.id,"",tdJSON)
        return this.sendNotification(msg)
    }

    // obtain a new token
    async refreshToken(): Promise<string> {

        let refreshPath = HttpPostRefreshPath.replace("{thingID}", "authn")
        refreshPath = refreshPath.replace("{name}", "refreshMethod")
        // TODO use generated API
        let args = {
            clientID: this.clientID,
            oldToken: this.authToken,
        }
        try {
            let resp = await this.pubMessage("POST",refreshPath, args,"");
            this.authToken = JSON.parse(resp)
            return this.authToken
        } catch (e) {
            hclog.error("refreshToken failed: ", e)
            throw e
        }
    }

    // sendNotification [agent] sends a notification message to the hub.
    // @param msg: notification to send
    sendNotification(notif: NotificationMessage) {
        // notifications can only be send to the fixed endpoint
        this.pubMessage(
            "POST",HiveOTPostNotificationHRef, notif)
            .then().catch()
    }

    // sendResponse [agent] sends a response status message to the hub.
    // @param resp: response to send
   async sendRequest(req: RequestMessage):Promise<ResponseMessage> {
        return new Promise((resolve, reject): void => {
            // use forms for requests for interoperability
            let href: string | undefined
            let input: any
            let f: TDForm | undefined
            if (this.getForm) {
                f = this.getForm(req.operation)
            }
            if (f) {
                href = f.getHRef()
            }
            if (href) {
                this.pubMessage("POST", href, req.input, req.correlationID)
                    .then((reply: string) => {
                        let output = JSON.parse(reply)
                        let resp = new ResponseMessage(
                            req.operation, req.thingID, req.name, output, "", "")
                        resolve(resp)
                    })
            } else {
                this.pubMessage("POST", HiveOTPostRequestHRef, input, req.correlationID)
                    .then((reply: string) => {
                        let resp: ResponseMessage = JSON.parse(reply)
                        resolve(resp)
                    })
            }
        })
    }

    // sendResponse [agent] sends a response status message to the hub.
    // @param resp: response to send
    sendResponse(resp: ResponseMessage) {
        // responses can only be send to the fixed endpoint
        this.pubMessage(
            "POST",HiveOTPostResponseHRef, resp,resp.correlationID)
            .then().catch()
    }


    // Rpc publishes an RPC request to a service and waits for a response.
    // Intended for users and services to invoke RPC to services.
    // This return the response data.
    async rpc(dThingID: string, methodName: string, args: any): Promise<any> {
        return new Promise((resolve, reject) => {

            // a correlationID is needed before the action is published in order to match it with the reply
            let correlationID = "rpc-" + nanoid()

            // handle timeout
            let t1 = setTimeout(() => {
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
                    resolve(resp.output)
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


    setConnectHandler(handler: (status: ConnectionStatus) => void): void {
        this.connectHandler = handler
    }
    setNotificationHandler(handler: NotificationHandler) {
        this.notificationHandler = handler
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
        let req = new RequestMessage({
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
        let req = new RequestMessage({
            operation:op, thingID:dThingID, name:name
        })
        this.sendRequest(req)
            .then().catch()
    }

    // writeProperty publishes a request for changing a Thing's property.
    // The configuration is a writable property as defined in the Thing's TD.
    writeProperty(thingID: string, name: string, propValue: any) {
        hclog.info("writeProperty. thingID:", thingID, ", name:", name)

        let req = new RequestMessage({
            operation: OpWriteProperty, thingID: thingID, name: name, input: propValue
        })
        return this.sendRequest(req)
    }
}
