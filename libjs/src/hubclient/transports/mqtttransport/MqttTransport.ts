
// MqttTransport
import { ConnInfo, ConnectionStatus } from "../IHubTransport";
import type { IHubTransport } from "../IHubTransport";
import * as mqtt from 'mqtt';
import * as os from "os";
import type { IHiveKey } from "@hivelib/keys/IHiveKey";
import { ECDSAKey } from "@hivelib/keys/ECDSAKey";
import type { IPublishPacket } from 'mqtt';
import type { QoS } from "mqtt-packet";
import * as tslog from 'tslog';

const log = new tslog.Logger()

export class MqttTransport implements IHubTransport {
    // expect mqtt://addr:port/
    fullURL: string
    clientID: string
    caCertPem: string
    instanceID: string = ""
    inboxTopic: string = ""
    timeoutMSec: number = 30000 // request timeout in millisec
    qos: QoS = 0

    // application handler of connection status change
    connectHandler: null | ((status: ConnectionStatus, info: string) => void) = null;
    // application handler of incoming messages
    eventHandler: null | ((topic: string, payload: string) => void) = null;
    // application handler of incoming request-response messages
    requestHandler: null | ((topic: string, payload: string) => string) = null;

    // myKeys:IHiveKey
    // https://github.com/mqttjs/MQTT.js/
    mcl: mqtt.MqttClient | null = null

    // map of correlationID to handler receiving a reply or timeout error
    replyHandlers: { [index: string]: (corrID: string, reply: string) => void };

    constructor(
        fullURL: string, clientID: string, caCertPem: string) {

        this.fullURL = fullURL
        this.clientID = clientID
        this.caCertPem = caCertPem
        this.replyHandlers = {}
    }

    public addressTokens(): { sep: string, wc: string, rem: string } {
        return { sep: "/", wc: "+", rem: "#" }
    }

    // connect and subscribe to the inbox
    public async connectWithPassword(password: string): Promise<void> {
        // let urlParts = new URL(this.fullURL)
        let p = new Promise<void>((resolve, reject) => {

            // log.info("connectWithPassword; url:", this.fullURL, "; clientID:", this.clientID)

            let timestamp = Date.now().toString() // msec since epoch
            let rn = Math.random().toString(16).substring(2, 8)
            this.instanceID = this.clientID + "-" + os.hostname + "-" + timestamp + "-" + rn
            // see https://github.com/mqttjs/MQTT.js/ for options
            let opts: mqtt.IClientOptions = {
                username: this.clientID,
                password: password,
                //
                ca: this.caCertPem,
                clean: true, // false
                clientId: this.instanceID,
                keepalive: 60,
                protocolVersion: 5, // MQTT 5 is required for rpc
                rejectUnauthorized: (this.caCertPem != ""),
                // properties: {
                //     userProperties: {
                //         "clientID": this.clientID
                //     }
                // }

            }

            this.mcl = mqtt.connect(this.fullURL, opts)

            this.mcl.on("connect", (packet: any) => {
                log.info("MQTT client connected to:", this.fullURL)
                // subscribe to inbox; server only allows inbox subscription with the clientID
                // to prevent subscribing to other client's inboxes.
                this.inboxTopic = "_INBOX/" + this.instanceID
                if (this.mcl) {
                    let inboxSub = this.mcl.subscribe(this.inboxTopic)
                    log.info("subscribed to ", this.inboxTopic)
                }
                resolve()
                if (this.connectHandler) {
                    this.connectHandler(ConnectionStatus.Connected, ConnInfo.Success)
                }
            })
            this.mcl.on("disconnect", (packet: mqtt.IDisconnectPacket) => {
                log.info("MQTT server disconnected:")
                let reason: string = packet.reasonCode?.toString() || ""
                let err: Error | null = null
                if (packet.reasonCode != 0) {
                    err = new Error(reason + ":" + packet.properties?.reasonString)
                }
                if (this.connectHandler) {
                    this.connectHandler(ConnectionStatus.Disconnected, err?.message || "")
                }
            })

            this.mcl.on("error", (err: Error) => {
                log.error("MQTT error: ", err.message)
                reject(err)
            })
            this.mcl.on("message",
                (topic: string, payload: Buffer, packet: IPublishPacket) => {
                    // log.info("on message. topic:", topic)
                    this.onRawMessage(topic, payload, packet)
                })
            this.mcl.on("offline", () => {
                log.error("Connection lost")
            })
            // this.mcl.on("packetreceive", (packet: mqtt.Packet) => {
            //     log.error("packetReceive: cmd:", packet.cmd)
            // })


        })
        return p
    }

    public async connectWithToken(key: IHiveKey, token: string): Promise<void> {
        // TODO: encrypt token with server public key so a MIM won't be able to get the token
        return this.connectWithPassword(token)
    }

    // this mqtt transport uses ECDSA keys
    public createKeyPair(): IHiveKey {
        let kp = new ECDSAKey().initialize()
        return kp
    }

    public disconnect(): void {
        if (this.mcl != null) {
            this.mcl.end()
            this.mcl = null
        }
    }

    // handle incoming mqtt message.
    // This determines the type of message and passes it on to the corresponding handlers.
    //  * INBOX message: The handler registered to the correlationID will be invoked.
    //  * Message with replyTo address: the RequestHandler will be invoked and its result is sent to the replyTo address.
    //  * Message with no replyTo address: The MessageHandler will be invoked.
    onRawMessage(topic: string, payload: Buffer, packet: mqtt.IPublishPacket): void {
        // intercept INBOX replies
        let payloadStr = payload.toString()
        let cordata = packet.properties?.correlationData
        // TODO: an inbox address could include the correlation ID. Is this worth it for mqtt-v3?
        let corid = cordata?.toString() || ""
        try {
            if (topic.startsWith(this.inboxTopic)) {
                // this is a response
                if (corid == "") {
                    log.error("onRawMessage: ignored response without correlation ID.")
                } else {
                    // find the handler for the correlation ID.
                    let handler = this.replyHandlers[corid]
                    handler(corid, payloadStr)
                    return
                }
            } else if (packet.properties?.responseTopic) {
                let err: Error | null = null
                let reply: string = ""
                // this is a request message that asks for a response
                if (!corid) {
                    err = Error("onRawMessage: request without correlationID. Topic:" + topic)
                } else if (!this.requestHandler) {
                    err = Error("onRawMessage: No handler for topic:" + topic)
                } else {
                    // this is a request that asks for a reponse
                    try {
                        reply = this.requestHandler(topic, payloadStr)
                    } catch (e: any) {
                        // catch this error so a reply can be sent
                        err = Error("onRawMessage ERROR: exception:", e)
                    }
                }
                this.pubReply(reply, err, packet)

            } else {
                // this is an event type message
                if (this.eventHandler) {
                    this.eventHandler(topic, payloadStr)
                }
            }
        } catch (e) {
            log.error("onRawMessage: Exception handling message. topic:", topic, ", err:", e)
        }
    }

    // send an event and return immediately
    public async pubEvent(address: string, payload: string): Promise<void> {
        if (this.mcl) {

            let opts: mqtt.IClientPublishOptions = {
                qos: this.qos,
            }
            // this.mcl.publish(address, payload)
            this.mcl.publish(address, payload, opts)
        }
        return
    }

    // send an action or RPC request and wait for a reply
    public async pubRequest(address: string, payload: string): Promise<string> {

        let p = new Promise<string>((resolve, reject) => {
            let rn = Math.random().toString(16).substring(2, 8)
            // let corid = Date.now().toString(16) + "." + rn
            let rightnow = new Date()
            let corid = rightnow.toISOString() + "." + rn
            let opts: mqtt.IClientPublishOptions = {
                qos: this.qos,
                properties: {
                    responseTopic: this.inboxTopic,
                    correlationData: Buffer.from(corid)
                }
            }
            this.mcl ? this.mcl.publish(
                address, payload, opts, (err) => {
                    if (err) {
                        log.info(err)
                        reject()
                        throw (err)
                    } else {
                        log.info('pubRequest: Request sent to ' + address + 'with correlationID:', corid)
                    }
                }) : "";
            // if the timer isn't cancelled in time, it will reject the request
            // this should work with both node and browser
            // FIXME remove reply handler after use
            let timoutID = setTimeout(() => reject("timeout"), this.timeoutMSec)
            this.replyHandlers[corid] =
                function (corrID: string, payload: string): void {
                    // received a reply, cancel the timer and resolve the request
                    clearTimeout(timoutID)
                    log.info("pubRequest: invoking reply handler")
                    resolve(payload)
                }
        })
        return p
    }

    // send a reply to a request
    async pubReply(payload: string, err: Error | null, request: IPublishPacket) {
        if ((!this.mcl) || (!request.properties?.correlationData) || !request.properties.responseTopic) {
            let err = Error("pubReply: missing replyTo or correlationData")
            log.error(err)
            throw err
        }
        let corid = request.properties?.correlationData
        // FIXME send as a reply without responseTopic
        let opts: mqtt.IClientPublishOptions = {
            qos: this.qos,
            properties: {
                correlationData: corid,
            }
        }
        // typescript doesn't recognize that opts.properties is already set
        if (err && (!!opts.properties)) {
            let userProp = { error: err.message };
            opts.properties.userProperties = userProp;
        }
        let replyTo = request.properties?.responseTopic
        this.mcl.publish(replyTo, payload, opts, (err) => {
            if (err) {
                // failed to send a reply
                log.error("pubReply: failed to send reply. err=", err)
                throw (err)
            } else {
                log.info('pubReply: Request sent with correlation ID:', corid.toString())
            }
        })
        return
    }

    // Set the callback of connect/disconnect updates
    public setConnectHandler(handler: (status: ConnectionStatus, info: string) => void): void {
        this.connectHandler = handler
    }

    // Set the handler of incoming messages
    public setEventHandler(handler: (topic: string, payload: string) => void): void {
        this.eventHandler = handler
    }

    // Set the handler of incoming requests-response calls.
    // The result of the handler is sent as a reply.
    // Intended for handling actions and RPC requests.
    public setRequestHandler(handler: (topic: string, payload: string) => string): void {
        this.requestHandler = handler
    }

    // subscribe to a topic
    public async subscribe(topic: string): Promise<void> {
        let p = new Promise<void>((resolve, reject) => {
            if (!this.mcl) {
                throw ("subscribe: no server connection");
            }
            // this.mcl.subscribe(topic, opts, (err, granted) => {
            let opts: mqtt.IClientSubscribeOptions = {
                qos: this.qos
            }
            this.mcl.subscribe(topic, opts, (err, granted) => {
                // remove registration if subscription fails
                if (err) {
                    log.error("subscribe: failed: " + err);
                    reject(err)
                } else {       // all good
                    log.info("subscribe: topic:" + topic);
                    resolve()
                }
            })
        })
        return p;

    }

    public unsubscribe(address: string) {
        if (this.mcl) {
            this.mcl.unsubscribe(address);
        }
    }
}

// Create a new MQTT transport using websockets over SSL
//
// fullURL schema supports: mqtt, mqtts, tcp, tls, ws, wss, wxs, alis
//
//
// @param fullURL is the websocket address: wss://address:port/path
// @param clientID is the client's connection ID
// @param caCertPem is the pem encoded CA certificate, if available. Use "".
// @param onMessage is the message handler for subscriptions
// export function NewMqttTransport(fullURL:string, clientID:string, caCertPem:string,
//                                  onMessage:(topic:string,msg:string)=>void): MqttTransport {
//     //
//     // caCertPool := x509.NewCertPool()
//     // if caCert != nil {
//     //     caCertPool.AddCert(caCert)
//     // }
//     // tlsConfig := &tls.Config{
//     //     RootCAs:            caCertPool,
//     //         InsecureSkipVerify: caCert == nil,
//     // }
//
//
//     let tp = new MqttTransport(fullURL,clientID,caCertPool,onMessage)
//     return tp
// }