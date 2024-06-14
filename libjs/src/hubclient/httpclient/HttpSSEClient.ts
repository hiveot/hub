import {TD} from '../../things/TD.js';
import {
    ConnectionStatus,
    ConnInfo,
    DeliveryProgress,
    DeliveryStatus,
    EventHandler,
    IHubClient,
    MessageHandler
} from "../IHubClient.js";
import {ThingMessage} from "../../things/ThingMessage";
import type {IHiveKey} from "@keys/IHiveKey";
import * as tslog from 'tslog';
import {
    ActionTypeProperties,
    EventTypeProperties,
    EventTypeTD,
    PostActionPath,
    PostEventPath,
    PostLoginPath, PostRefreshPath,
    PostSubscribePath,
    PostUnsubscribePath
} from "@hivelib/api/vocab/ht-vocab";
import * as http2 from "node:http2";
import {string} from "yaml/dist/schema/common/string";


const log = new tslog.Logger()

// HubClient implements the javascript client for connecting to the hub
// using HTTPS and SSE for the return channel.
export class HttpSSEClient implements IHubClient {
    _clientID: string;
    _baseURL: string;
    _caCertPem: string;
    _disableCertCheck: boolean
    _http2Client: http2.ClientHttp2Session|undefined;

    isInitialized: boolean = false;
    connStatus: ConnectionStatus;
    connInfo: string;
    // the auth token when using connectWithToken
    authToken: string|undefined;

    // client handler for action requests of things published by this client, if any.
    actionHandler: MessageHandler | null = null;
    // client handler for connection status change
    connectHandler: ((status: ConnectionStatus, info: ConnInfo) => void) | null = null
    // client handler for subscribed events
    eventHandler: EventHandler | null = null;

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
        this._baseURL = hostPort
        this._clientID = clientID;
        this._caCertPem = caCertPem;
        this._disableCertCheck = disableCertCheck;
        this.connStatus = ConnectionStatus.Disconnected;
        this.connInfo = ConnInfo.NotConnected;
    }

    // ClientID the client is authenticated as to the server
    get clientID(): string {
        return this._clientID;
    }

    // return the current connection status
    get connectionStatus(): { status: ConnectionStatus, info: string } {
        return {status: this.connStatus, info: this.connInfo};
    }

    // setup a TLS connection with the hub
    async connect():Promise<http2.ClientHttp2Session> {

        if (this._disableCertCheck) {
            log.warn("Disabling server certificate check.")
        }
        let opts:  http2.SecureClientSessionOptions = {
            timeout: 10000, // msec???
            "rejectUnauthorized": !this._disableCertCheck
        }
        if (!!this._caCertPem) {
            opts.ca = this._caCertPem
        }
        this._http2Client = http2.connect(this._baseURL, opts)

        // When an error occurs, show it.
        this._http2Client.on('error', (error)=> {
            console.error(error);
            // this.disconnect()
            // Close the connection after the error occurred.
            // client.destroy();
        });
        this._http2Client.on('end', () => {
            console.log('server ends connection');
            this.disconnect()
        });

        return this._http2Client
    }

    // ConnectWithPassword connects to the Hub server using the clientID and password.
    async connectWithPassword(password: string):Promise<string> {
        let token: string = ""
        // establish a session
        await this.connect()
        // invoke a login request
        let loginArgs = {
            "clientID":this._clientID,
            "password": password,
        }
        let loginResp = {
            sessionID: "",
            token: ""
        }
        let loginMsg = JSON.stringify(loginArgs)
        let resp = await this.postRequest(PostLoginPath,loginMsg)
        loginResp = JSON.parse(resp)
        this.authToken = loginResp.token
        return loginResp.token
    }
    // connect and login to the Hub gateway using a JWT token
    // host is the server address
    async connectWithToken(jwtToken: string):Promise<string> {
        this.authToken=jwtToken
        await this.connect()
        return ""
    }

    createKeyPair(): IHiveKey|undefined {
        // FIXME:todo
        return
    }

    // disconnect if connected
    async disconnect() {
        if (this._http2Client) {
            this._http2Client.close();
            this._http2Client = undefined
        }
        if (this.connStatus != ConnectionStatus.Disconnected) {
            this.connStatus = ConnectionStatus.Disconnected
        }
    }

    // callback handler invoked when the SSE connection status has changed
    onConnection(status: ConnectionStatus, info: string) {
        this.connStatus = status
        this.connInfo = info
        if (this.connStatus == ConnectionStatus.Connected) {
            log.info('HubClient connected');
        } else if (this.connStatus == ConnectionStatus.Connecting) {
            log.info('HubClient attempt connecting');
        } else {
            log.info('HubClient disconnected');
        }
    }

    //
    // // Handle incoming messages and pass them to the event handler
    // onEvent(addr: string, payload: string): void {
    //     let {msgType, agentID, thingID, name, senderID, err} =
    //         this._splitAddress(addr)
    //     let timestampMsec = Date.now() // UTC in msec
    //     let tv: ThingMessage = {
    //         agentID: agentID,
    //         thingID: thingID,
    //         key: name,
    //         senderID: senderID,
    //         valueType: msgType,
    //         createdMSec: timestampMsec,
    //         data: payload,
    //     }
    //     if (err != null) {
    //         err = new Error("handleEvent: Received event on invalid address '" + addr + "': " + err.message)
    //         log.info(err)
    //         throw err
    //     }
    //     if (this.eventHandler != null) {
    //         this.eventHandler(tv)
    //     }
    // }

    // // Handle incoming action or config request messages and pass them on to their
    // // respective handlers, if set.
    // // The response will be sent back to the caller.
    // onRequest(addr: string, payload: string): string {
    //
    //     let {msgType, agentID, thingID, name, senderID, err} =
    //         this._splitAddress(addr)
    //     let timestampMsec = Date.now() // UTC in msec
    //
    //     let tv: ThingMessage = {
    //         agentID: agentID,
    //         thingID: thingID,
    //         key: name,
    //         senderID: senderID,
    //         valueType: msgType,
    //         createdMSec: timestampMsec,
    //         data: payload,
    //     }
    //     if (senderID == "") {
    //         err = new Error("handleRequest: Missing senderID on address '" + addr + ", request ignored.")
    //         log.info(err)
    //         throw err
    //     } else if (err != null) {
    //         err = new Error("handleRequest: Received request on invalid address '" + addr + "': " + err.message)
    //         log.info(err)
    //         throw err
    //     }
    //     // detection of request must have been mistaken as only subscriptions with this
    //     // clientID are made.
    //     if (agentID != this._clientID) {
    //         err = new Error("request received for another agent");
    //         log.error(err)
    //         throw err
    //     }
    //
    //     if (msgType == MessageTypeAction && this.actionHandler != null) {
    //         return this.actionHandler(tv)
    //     } else if (msgType == MessageType.Config && this.configHandler != null) {
    //         let success = this.configHandler(tv)
    //         if (!success) {
    //             err = new Error("handleRequest: Config request not accepted")
    //             log.info(err)
    //             throw err
    //         } else {
    //             return ""
    //         }
    //     } else {
    //         err = Error("handleRequest: No handler is set for " + msgType + " messages")
    //         throw err
    //     }
    // }


    // post a request to the path with the given payload
    async postRequest(path:string, payload: string):Promise<string> {
        return new Promise((resolve,reject)=> {
            let replyData: string = ""

            if (!this._http2Client) {
                throw ("not connected")
            }
            let req = this._http2Client.request({
                origin: this._baseURL,
                authorization: "bearer "+this.authToken,
                ':path': path,
                ":method": "POST",
                "content-type": "application/json",
                "content-length": Buffer.byteLength(payload),
            })
            req.setEncoding('utf8');

            req.on('data', (chunk) => {
                replyData = replyData + chunk
            });
            req.on('end', () => {
                req.destroy()
                log.info(`postRequest to ${path}. Received reply. size=`+ replyData.length)
                resolve(replyData)
            });
            req.on('error', (err) => {
                reject(err)
                req.destroy()
            });
            // write the body and complete the request
            req.end(payload)
        });
    }

    // PubAction publishes a request for action from a Thing.
    //
    //	@param agentID: of the device or service that handles the action.
    //	@param thingID: is the destination thingID to whom the action applies.
    //	name is the name of the action as described in the Thing's TD
    //	payload is the optional serialized message of the action as described in the Thing's TD
    //
    // This returns the serialized reply data or null in case of no reply data
    async pubAction(thingID: string, key: string, payload: string): Promise<DeliveryStatus> {
            log.info("pubAction. thingID:", thingID, ", key:", key)

            let actionPath = PostActionPath.replace("{thingID}", thingID)
            actionPath = actionPath.replace("{key}", key)

            let resp = await this.postRequest(actionPath, payload)
            let stat: DeliveryStatus = JSON.parse(resp)
            return stat
    }

    // PubAction publishes a request for changing a Thing's configuration.
    // The configuration is a writable property as defined in the Thing's TD.
    async pubConfig(thingID: string, propName: string, propValue: string): Promise<DeliveryStatus> {
        let props = {propName:propValue}
        let propsJson = JSON.stringify(props)
        return  this.pubAction(thingID, ActionTypeProperties, propsJson)
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
    async pubEvent(thingID: string, key: string, payload: string):Promise<DeliveryStatus> {
        let eventPath = PostEventPath.replace("{thingID}",thingID)
        eventPath = eventPath.replace("{key}",key)

        let resp = await this.postRequest(eventPath, payload)
        let stat: DeliveryStatus = JSON.parse(resp)
        return stat
    }

    // Publish a Thing properties event
    async pubProps(thingID: string, props: {[key:string]:string}): Promise<DeliveryStatus> {
        // if (length(props.) > 0) {
        let propsJSON = JSON.stringify(props, null, ' ');
         return this.pubEvent(thingID, EventTypeProperties, propsJSON);
    }


    // Rpc publishes an RPC request to a service and waits for a response.
    // Intended for users and services to invoke RPC to services.
    async rpc(dThingID: string, methodName: string, args: any): Promise<any> {

        let payload = JSON.stringify(args)
        let stat = await this.pubAction(dThingID,methodName, payload);
        if (stat.error != "") {
            throw stat.error
        }
        // TODO: wait for status update reply
        return stat
    }

    // PubTD publishes an event with a Thing TD document.
    // The client's authentication ID will be used as the agentID of the event.
    async pubTD(td: TD) {
        let tdJSON = JSON.stringify(td, null, ' ');
        return this.pubEvent(td.id, EventTypeTD, tdJSON);
    }


    // obtain a new token
    async refreshToken(): Promise<string> {

        let refreshPath = PostRefreshPath.replace("{thingID}","authn")
        refreshPath = refreshPath.replace("{key}","refreshMethod")
        let resp =  await this.postRequest(refreshPath,"");
        let stat: DeliveryStatus = JSON.parse(resp)
        let newToken = stat?.reply ? stat?.reply : ""
        this.authToken = newToken
        return newToken
    }

    // set the handler of thing action requests and subscribe to action requests
    setActionHandler(handler: MessageHandler) {
        this.actionHandler = handler
    }

    setConnectHandler(handler: (status: ConnectionStatus, info: string) => void): void {
        this.connectHandler = handler
    }

    // set the handler for subscribed events
    setEventHandler(handler: EventHandler): void {
        this.eventHandler = handler
    }

    // set the handler of rpc requests
    // setRPCHandler(handler: (tv: ThingMessage) => string) {
    //     this.rpcHandler = handler
    //     // handler of requests is the same for actions, config and rpc
    //     this.tp.setRequestHandler(this.onRequest.bind(this))
    //     let addr = this._makeAddress(MessageType.RPC, this.clientID, "", "", "")
    //     this.tp.subscribe(addr)
    //
    // }


    // Read Thing definitions from the directory
    // @param publisherID whose things to read or "" for all publishers
    // @param thingID whose to read or "" for all things of the publisher(s)
    // async readDirectory(agentID: string, thingID: string): Promise<string> {
    // 	return global.hapiReadDirectory(publisherID, thingID);
    // }

    // Subscribe to events from things.
    //
    // The events will be passed to the configured onEvent handler.
    //
    // note there is no unsubscribe. The intended use is to subscribe to devices/things/events
    // of interest and leave it at that. Currently there is no use-case that requires
    // a frequent subscribe/unsubscribe.
    //
    // @param agentID: optional filter on the agent that publishes events; "" for all agents.
    // @param thingID: optional filter of the thing whose events are published; "" for all things
    // @param eventID: optional filter on the event name; "" for all event names.
    async subscribe(dThingID: string, key: string): Promise<void> {

        let subscribePath = PostSubscribePath.replace("{thingID}",dThingID)
        subscribePath = subscribePath.replace("{key}",key)
        await this.postRequest(subscribePath,"")
    }

    async unsubscribe(dThingID: string) {

        let subscribePath = PostUnsubscribePath.replace("{thingID}",dThingID)
        subscribePath = subscribePath.replace("{key}","+")
        await this.postRequest(subscribePath,"")
    }

}
