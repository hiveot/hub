import type { IHiveKey } from "@hivelib/keys/IHiveKey";
import type { ConnectionStatus, IHubTransport } from "../IHubTransport";
import * as tslog from 'tslog';
import type { SubscriptionOptions, ConnectionOptions, NatsConnection } from "nats.ws";
import { connect, nkeyAuthenticator, nkeys, StringCodec } from "nats.ws";
import { natsKey } from "@hivelib/keys/natsKey";

const log = new tslog.Logger()


export class NatsTransport implements IHubTransport {
    // expect nats://addr:port/ 
    fullURL: string
    clientID: string
    caCertPem: string
    instanceID: string = ""
    nc: NatsConnection | undefined

    // application handler of connection status change
    connectHandler: null | ((connectStatus: ConnectionStatus, info: string) => void) = null;
    // application handler of incoming messages
    eventHandler: null | ((topic: string, payload: string) => void) = null;
    // application handler of incoming request-response messages
    requestHandler: null | ((topic: string, payload: string) => string) = null;

    constructor(
        fullURL: string, clientID: string, caCertPem: string) {

        this.fullURL = fullURL
        this.clientID = clientID
        this.caCertPem = caCertPem
    }

    public addressTokens(): { sep: string, wc: string, rem: string } {
        return { sep: ".", wc: "*", rem: ">" }
    }

    // connect and subscribe to the inbox
    public async connectWithPassword(password: string): Promise<void> {
        let opts: ConnectionOptions = {
            servers: this.fullURL,
            user: this.clientID,
            pass: password,
        }
        this.nc = await connect(opts)
        return
    }

    // connectWithToken connects using an nkey public key.
    // jwt isn't yet supported as it requires a callout server, which is experimental.
    public async connectWithToken(key: IHiveKey, token: string): Promise<void> {
        let seed = key.exportPrivate()
        let seedUint8 = new TextEncoder().encode(seed)
        let opts: ConnectionOptions = {
            servers: this.fullURL,
            user: this.clientID,
            token: token,
            authenticator: nkeyAuthenticator(seedUint8)
        }
        this.nc = await connect(opts)
    }

    // the nats transport uses nkey keys
    public createKeyPair(): IHiveKey {
        let kp = new natsKey()
        return kp
    }

    public disconnect(): void {
        if (!!this.nc) {
            this.nc.close()
            // await this.nc.closed()
            this.nc = undefined
        }
    }



    public async pubEvent(address: string, payload: string): Promise<void> {
        if (!this.nc) {
            throw ("pubEvent: not connected")
        }
        return await this.nc.publish(address, payload)
    }



    public async pubRequest(address: string, payload: string): Promise<string> {
        if (!this.nc) {
            throw ("pubRequest: not connected")
        }
        let resp = await this.nc.request(address, payload,)

        // error responses are stored in the header
        if (resp.headers) {
            let errMsg = resp.headers.get("error")
            if (errMsg != "") {
                let err = new Error(errMsg)
                throw (err)
            }
        }

        return resp.data.toString()
    }


    // Set the callback of connect/disconnect updates
    public setConnectHandler(handler: (connectStatus: ConnectionStatus, info: string) => void) {
        this.connectHandler = handler
    }

    // Set the handler of incoming messages
    public setEventHandler(handler: (topic: string, payload: string) => void) {
        this.eventHandler = handler
    }

    // Set the handler of incoming requests-response calls.
    // The result of the handler is sent as a reply.
    // Intended for handling actions and RPC requests.
    public setRequestHandler(handler: (topic: string, payload: string) => string) {
        this.requestHandler = handler
    }

    // subscribe to a NATS subject
    public async subscribe(address: string): Promise<void> {

        const sc = StringCodec();

        let nsub = await this.nc?.subscribe(address)
        if (!nsub || nsub.isClosed()) {
            throw ("failed to subscribe: ")
        }
        for await (const m of nsub) {
            log.info(`[${nsub.getProcessed()}]: ${m.subject}: ${sc.decode(m.data)}`);
            if (this.eventHandler) {
                let dataStr = sc.decode(m.data)
                this.eventHandler(m.subject, dataStr)
            }
        }
    }
    public unsubscribe(address: string) {
        // on-the-fly subscribe-unsubscribe is not the intended use.
        // if there is a good use-case it can be added, but it would
        // mean tracking the subscriptions.
        log.warn("unsubscribe is not used", "topic", address)
    }
}