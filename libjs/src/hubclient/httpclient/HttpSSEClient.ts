import {TD} from '../../things/TD.js';
import {
    IAgentClient,
} from "../IAgentClient.js";
import type {IHiveKey} from "@keys/IHiveKey";
import * as tslog from 'tslog';
import {
     MessageTypeProgressUpdate,
    MessageTypeAction, MessageTypeEvent, MessageTypeProperty,
} from "@hivelib/api/vocab/vocab.js";
import * as http2 from "node:http2";
import {connectSSE} from "@hivelib/hubclient/httpclient/connectSSE";
import {ThingMessage} from "@hivelib/things/ThingMessage";
import * as https from "node:https";
import {RequestProgress} from "@hivelib/hubclient/RequestProgress";
import {
    ActionHandler,
    ConnectionStatus,
    EventHandler,
    ProgressHandler,
    PropertyHandler
} from "@hivelib/hubclient/IConsumerClient";
import {nanoid} from "nanoid";
import EventSource from "eventsource";

// FIXME: import from vocab is not working
const RequestCompleted = "completed"
const RequestFailed = "failed"


// Form paths that apply to all TDs at the top level
// SYNC with HttpSSEClient.go
// These are intended for use by agents - use forms instead
//
// FIXME: THESE WILL BE REMOVED WHEN SWITCHING TO FORMS
//
const ConnectSSEPath      = "/ssesc"
const PostSubscribeEventPath   = "/ssesc/digitwin/subscribe/{thingID}/{name}"
const PostUnsubscribeEventPath = "/ssesc/digitwin/unsubscribe/{thingID}/{name}"

// paths for accessing TDD directory
// const GetThingPath= "/digitwin/directory/{thingID}"
// const GetAllThingsPath= "/digitwin/directory" // query param offset=, limit=
// const PostThingPath    = "/agent/tdd/{thingID}"

// paths for accessing actions
const PostInvokeActionPath   = "/digitwin/actions/{thingID}/{name}"

// paths for use by agents - TODO: agents should use forms from the digitwin agent TD
const PostAgentPublishEventPath    = "/agent/event/{thingID}/{name}"
const PostAgentPublishProgressPath  = "/agent/progress"
const PostAgentUpdatePropertyPath   = "/agent/property/{thingID}/{name}"
const PostAgentUpdateMultiplePropertiesPath = "/agent/properties/{thingID}"
const PostAgentUpdateTDDPath                = "/agent/tdd/{thingID}"

// paths for accessing properties - TODO: MUST USE FORMS
const PostWritePropertyPath = "/digitwin/properties/{thingID}/{name}"

// authn service - used in authn
const PostLoginPath   = "/authn/login"
const PostLogoutPath  = "/authn/logout"
const PostRefreshPath = "/authn/refresh"


const hclog = new tslog.Logger()

// HubClient implements the javascript client for connecting to the hub
// using HTTPS and SSE for the return channel.
export class HttpSSEClient implements IAgentClient {
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
    connectHandler: ((status: ConnectionStatus) => void) | null = null
    // action handler for incoming messages from the hub.
    actionHandler: ActionHandler | null = null;
    // event handler
    eventHandler: EventHandler | null = null;
    // property write request (as agent) or property update (as consumer) handler
    propertyHandler: PropertyHandler | null = null;
    // progress handler receiving progress updates from agents
    progressHandler: ProgressHandler | null = null;

    // map of requestID to delivery status update channel
    _correlData: Map<string,(stat: RequestProgress)=>void>

    // Instantiate the Hub Client.
    //
    // The flag disableCertCheck is intended for use with self-signed certificate
    // on the local network where the CA is not available.
    //
    // @param hostPort: of server to connect to or empty for auto-discovery
    // @param clientID: connected as this client
    // @param caCertPem: verify server against this CA certificate
    // @param disableCertCheck: don't check the (self signed) server certificate
    constructor(hostPort: string, clientID: string, caCertPem: string, disableCertCheck: boolean) {
        this._baseURL = hostPort;
        this._caCertPem = caCertPem;
        this._clientID = clientID;
        this._disableCertCheck = disableCertCheck;
        this._ssePath = ConnectSSEPath;
        this.connStatus = ConnectionStatus.Disconnected;
        this.authToken = "";
        this._correlData = new Map();
        this._cid = nanoid() // connection id
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
            "clientID": this._clientID,
            "password": password,
        }
        let loginResp = {
            sessionID: "",
            token: ""
        }
        let resp = await this.pubMessage("POST", PostLoginPath,"", loginArgs)
        loginResp = JSON.parse(resp)
        this.authToken = loginResp.token
        // with the new auth token a SSE return channel can be established
        this._sseClient = await connectSSE(
            this._baseURL, this._ssePath, this.authToken, this._cid,
            this.onMessage.bind(this),
            this.onProgress.bind(this),
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
            this.onMessage.bind(this),
            this.onProgress.bind(this),
            this.onConnection.bind(this) )
        return ""
    }

    createKeyPair(): IHiveKey | undefined {
        // FIXME:todo
        return
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
    onProgress(stat:RequestProgress):void{
        let cb = this._correlData.get(stat.requestID)
        if (cb) {
            cb(stat)
        } else if (this.progressHandler ) {
            this.progressHandler(stat)
        }
    }

    // Handle incoming event or property messages from the hub and pass them to handler
    onMessage(msg: ThingMessage): void {
        try {
            if (msg.messageType == MessageTypeAction && this.actionHandler) {
                this.actionHandler(msg)
            } else if (msg.messageType == MessageTypeEvent && this.eventHandler) {
                this.eventHandler(msg)
            } else if (msg.messageType == MessageTypeProperty && this.propertyHandler) {
                 this.propertyHandler(msg)
            } else {
                hclog.warn(`onMessage unknown message type: ${msg.messageType}`)
            }
        } catch (e) {
            let errText = `Error handling hub message sender=${msg.senderID}, messageType=${msg.messageType}, thingID=${msg.thingID}, name=${msg.name}, error=${e}`
            hclog.warn(errText)
            if (msg.messageType == MessageTypeAction) {
                let stat = new RequestProgress()
                stat.failed(msg, errText)
                this.pubProgressUpdate(stat)
            }
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
    async pubMessage(methodName: string, path: string, requestID:string, data: any):Promise<string> {
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
                    "message-id": requestID,
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
    //
    //	@param agentID: of the device or service that handles the action.
    //	@param thingID: is the destination thingID to whom the action applies.
    //	name is the name of the action as described in the Thing's TD
    //  requestID to include in the header
    //	payload is the optional action arguments to be serialized and transported
    //
    // This returns the serialized reply data or null in case of no reply data
    async invokeAction(thingID: string, name: string, requestID:string,
                       payload: any): Promise<RequestProgress> {

        hclog.info("pubAction. thingID:", thingID, ", name:", name)

        let actionPath = PostInvokeActionPath.replace("{thingID}", thingID)
        actionPath = actionPath.replace("{name}", name)

        let resp = await this.pubMessage("POST",actionPath, requestID, payload)
        let stat: RequestProgress = JSON.parse(resp)
        return stat
    }


    // PubEvent publishes a Thing event. The payload is an event value as per TD document.
    // Intended for devices and services to notify of changes to the Things they are the agent for.
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
    pubEvent(thingID: string, name: string, payload: any) {
        hclog.info("pubEvent. thingID:", thingID, ", name:", name)

        let eventPath = PostAgentPublishEventPath.replace("{thingID}", thingID)
        eventPath = eventPath.replace("{name}", name)

        this.pubMessage("POST",eventPath, "", payload)
            .catch((e)=> {
                hclog.warn("failed publishing event", e)
            }
        )
    }


    // pubProgressUpdate sends a delivery status update back to the sender of the action
    // @param msg: action message that was received
    // @param stat: status to return
    pubProgressUpdate(stat: RequestProgress) {
        this.pubMessage(
            "POST",PostAgentPublishProgressPath, stat.requestID, stat)
            .then().catch()
    }

    // Publish batch of property values
    pubMultipleProperties(thingID: string, propMap: { [key: string]: any }) {
        let postPath = PostAgentUpdateMultiplePropertiesPath.replace("{thingID}", thingID)

        this.pubMessage("POST",postPath, "", propMap)
            .then().catch()
    }

    // Publish thing property value update
    pubProperty(thingID: string, name:string, value: any) {
        let postPath = PostAgentUpdatePropertyPath.replace("{thingID}", thingID)
         postPath = postPath.replace("{name}", name)

        this.pubMessage("POST",postPath, name, value)
            .then().catch()
    }

    // PubTD publishes an event with a Thing TD document.
    // This serializes the TD into JSON as per WoT specification
    pubTD(td: TD) {
        // FIXME: use action on the directory?
        let tdJSON = JSON.stringify(td, null, ' ');

        let postPath = PostAgentUpdateTDDPath.replace("{thingID}", td.id)
        this.pubMessage("POST",postPath, "", tdJSON)
            .then().catch()
    }


    // Rpc publishes an RPC request to a service and waits for a response.
    // Intended for users and services to invoke RPC to services.
    // This return the response data.
    async rpc(dThingID: string, methodName: string, args: any): Promise<any> {
        return new Promise((resolve, reject) => {

            // a requestID is needed before the action is published in order to match it with the reply
            let requestID = "rpc-" + nanoid()

            // handle timeout
            let t1 = setTimeout(() => {
                this._correlData.delete(requestID)
                console.error("RPC",dThingID,methodName,"failed with timeout")
                reject("timeout")
            }, 30000)

            // set the handler for progress messages
            this._correlData.set(requestID, (stat:RequestProgress):void=> {
                // console.log("delivery progress",stat.progress)
                // Remove the rpc wait hook and resolve the rpc
                clearTimeout(t1)
                this._correlData.delete(requestID)
                resolve(stat.reply)
            })
            this.invokeAction(dThingID, methodName, requestID, args)
                .then((stat: RequestProgress) => {
                    // complete the request if the result is returned, otherwise wait for
                    // the callback from _correlData
                    if (stat.progress == RequestCompleted || stat.progress == RequestFailed) {
                        this._correlData.delete(requestID)
                        resolve(stat.reply)
                    }
                })
                .catch((e) => {
                    console.error("RPC failed", e);
                    reject(e)
                })
        })
    }

    async waitForResponse(requestID:string): Promise<RequestProgress> {
        let stat = new RequestProgress()
        stat.progress = RequestFailed
        stat.error = "no response"
        return stat
    }

    // Read Thing definitions from the directory
    // @param publisherID whose things to read or "" for all publishers
    // @param thingID whose to read or "" for all things of the publisher(s)
    // async readDirectory(agentID: string, thingID: string): Promise<string> {
    // 	return global.hapiReadDirectory(publisherID, thingID);
    // }


    // obtain a new token
    async refreshToken(): Promise<string> {

        let refreshPath = PostRefreshPath.replace("{thingID}", "authn")
        refreshPath = refreshPath.replace("{name}", "refreshMethod")
        // TODO use generated API
        let args = {
            clientID: this.clientID,
            oldToken: this.authToken,
        }
        try {
            let resp = await this.pubMessage("POST",refreshPath, "", args);
            this.authToken = JSON.parse(resp)
            return this.authToken
        } catch (e) {
            hclog.error("refreshToken failed: ", e)
            throw e
        }
    }

    setConnectHandler(handler: (status: ConnectionStatus) => void): void {
        this.connectHandler = handler
    }

    // set the handler of incoming messages such as action requests or events
    //
    // The handler should return a DeliveryStatus containing the delivery progress.
    // This is ignored for events.
    //
    // Event messages are not received until the subscribe method is invoked with
    // the event keys to subscribe to.
    setActionHandler(handler: ActionHandler) {
        this.actionHandler = handler
    }
    setEventHandler(handler: EventHandler) {
        this.eventHandler = handler
    }
    setPropertyHandler(handler: PropertyHandler) {
        this.propertyHandler = handler
    }
    setProgressHandler(handler: ProgressHandler) {
        this.progressHandler = handler
    }

    // Subscribe to events from things.
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
        if (!dThingID) {
            dThingID = "+"
        }
        if (!name) {
            name = "+"
        }
        // FIXME: a connectionID is required for subscriptions over SSE
        let subscribePath = PostSubscribeEventPath.replace("{thingID}", dThingID)
        subscribePath = subscribePath.replace("{name}", name)
        this.pubMessage("POST", subscribePath,"", "")
            .then().catch()
    }

    unsubscribe(dThingID: string, name: string) {
        if (!dThingID) {
            dThingID = "+"
        }
        if (!name) {
            name = "+"
        }
        let subscribePath = PostUnsubscribeEventPath.replace("{thingID}", dThingID)
        subscribePath = subscribePath.replace("{name}", name)
        this.pubMessage("POST", subscribePath, "", "")
            .then().catch()
    }

    // writeProperty publishes a request for changing a Thing's property.
    // The configuration is a writable property as defined in the Thing's TD.
    writeProperty(thingID: string, name: string, propValue: any) {
        hclog.info("writeProperty. thingID:", thingID, ", name:", name)

        // TODO: use url from TD forms
        let propPath = PostWritePropertyPath.replace("{thingID}", thingID)
        propPath = propPath.replace("{name}", name)

        this.pubMessage("POST",propPath, "",propValue)
            .then().catch()
    }
}
