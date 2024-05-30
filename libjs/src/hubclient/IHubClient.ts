// ISubscription interface to underlying subscription mechanism
import type {IHiveKey} from "@keys/IHiveKey";
import {ThingMessage} from "@hivelib/things/ThingMessage";
import {TD} from "@hivelib/things/TD";


export enum ConnectionStatus {
    Connected = "connected",
    Connecting = "connecting",
    Disconnected = "disconnected",
}

export enum ConnInfo {
    // connection successful
    Success = "success",
    // ConnInfoUnauthorized credentials invalid
    Unauthorized = "unauthorized",
    // ConnInfoUnreachable unable to reach the server during the initial connection attempt
    Unreachable = "unreachable",
    // ConnInfoServerDisconnected a server disconnect message was received.
    ServerDisconnected = "serverDisconnected",
    // NetworkDisconnected connection has dropped. Caused by disconnecting the network somewhere
    NetworkDisconnected = "networkDisconnected",
    // NotConnected means the client has yet to connect
    NotConnected = "notConnected"
}

export enum DeliveryProgress {
    DeliveryCompleted = "completed",
    DeliveryDelivered = "delivered",
    DeliveryFailed = "failed",
}

// DeliveryStatus holds the progress of action request delivery
export class DeliveryStatus extends Object{
    // Request ID
    messageID: string = ""
    // Updated delivery status
    status: DeliveryProgress|undefined
    // Error in case delivery status is failed
    error?: string =""
    // Reply in case delivery status is completed
    reply?: string =""
}

// export class HubTransportStatus extends Object {
//     // URL of the hub
//     hubURL: string
//     // CA used to connect
//     caCert: *x509.Certificate
//     // the client ID to identify as
//     clientID: string
//
//     // The current connection status
//     connectionStatus: ConnectionStatus
//     // The last connection error message, if any
//     lastError: string//error
//
//     // flags indicating the supported protocols
//     supportsCertAuth:     boolean
//     supportsPasswordAuth: boolean
//     supportsKeysAuth:     boolean
//     supportsTokenAuth:    boolean
// }

// MessageTypeINBOX special inbox prefix for RPCs
// reserved event and action names
// export enum MessageType {
//     Action = "action",
//     Config = "config",
//     Event = "event",
//     RPC = "rpc",
//     INBOX = "_INBOX",
//     TD = "$td",
//     Props = "$properties"
// }

export type EventHandler = (msg:ThingMessage)=>void;

export type MessageHandler = (msg:ThingMessage)=>DeliveryStatus;

// IHubClient defines the interface of the hub transport client.
export interface IHubClient {
    // addressTokens returns the address separator and wildcard tokens used by the transport
    // @result sep is the address separator. eg "." for nats, "/" for mqtt and redis
    // @result wc is the address wildcard. "*" for nats, "+" for mqtt
    // @result rem is the address remainder. "" for nats; "#" for mqtt
    // addressTokens(): { sep: string, wc: string, rem: string };

    // ConnectWithPassword connects to the hub using password authentication.
    // @param password is created when registering the user with the auth service.
    // This returns an authentication token that can be used in refresh and connectWithToken.
    connectWithPassword(password: string): Promise<string>;

    // ConnectWithToken connects to the messaging server using an authentication token
    // and pub/private keys provided when creating an instance of the hub client.
    // @param token is created by the auth service.
    connectWithToken(token: string): Promise<string>;

    // CreateKeyPair returns a new key for authentication and signing.
    // @returns key contains the public/private key pair.
    createKeyPair(): IHiveKey|undefined;

    // Disconnect from the message bus.
    disconnect(): void;

    // PubAction publishes an action request and returns as soon as the request is delivered
    // to the Hub inbox.
    //
    //	@param dThingID the digital twin ID for whom the action is intended
    //	@param key is the action ID or method name of the action to invoke
    //  @param payload with serialized message to publish
    pubAction(thingID: string, key: string, payload: string): Promise<DeliveryStatus>;

    // getStatus returns the current transport connection status
    // getStatus(): HubTransportStatus

    // PubEvent publishes an event style message without a response.
    // It returns as soon as delivery to the hub is confirmed.
    //
    // Events are published by agents using their native ID, not the digital twin ID.
    // The Hub outbox broadcasts this event using the digital twin ID.
    //
    //	thingID native ID of the thing whose event is published
    //	key ID of the event
    //	payload with serialized message to publish
    //
    // This throws an error if the event cannot not be delivered to the hub
    pubEvent(thingID: string, key:string, payload: string): Promise<DeliveryStatus>;


    // PubProps publishes a property value map event.
    // It returns as soon as delivery to the hub is confirmed.
    // This is intended for agents, not for consumers.
    //
    // @param thingID is the ID of the device (not including the digital twin ID)
    // @param props is the property key-value map to publish where value is the serialized representation
    //
    // This throws an error if the event cannot not be delivered to the hub
    pubProps(thingID: string, props: Map<string,string>): Promise<DeliveryStatus>;

    // PubTD publishes an TD document event.
    // It returns as soon as delivery to the hub is confirmed.
    // This is intended for agents, not for consumers.
    //
    // @param td is the Thing Description document describing the Thing
    pubTD(td: TD): Promise<DeliveryStatus>

    // RefreshToken refreshes the authentication token
    // The resulting token can be used with 'ConnectWithJWT'
    refreshToken(): Promise<DeliveryStatus>

    // Rpc makes a RPC call using an action and waits for a delivery confirmation event.
    //
    // This is equivalent to use PubAction to send the request, use SetMessageHandler
    // to receive the delivery confirmation event and match the 'messageID' from the
    // delivery status event with the status returned by the action request.
    //
    // The arguments and responses are defined in structs (same approach as gRPC) which are
    // defined in the service api. This struct can also be generated from the TD document
    // if available at build time. See cmd/genapi for the CLI.
    //
    //	dThingID is the digital twin ID of the service providing the RPC method
    //	key is the ID of the RPC method as described in the service TD action affordance
    //	args is the address of a struct containing the arguments to marshal
    //
    // This returns the data or throws an error if failed
    rpc(dThingID: string, key: string, args: any, reply: any): Promise<any>

    // Set the handler for incoming action request.
    setActionHandler(cb: MessageHandler):void


    // set handler that is notified of changes in connection status and an error in
    // case of an  unintentional disconnect.
    // 
    // This handler is intended for updating presentation of the connection status.
    // Do not call connectXyz() in this handler, as a reconnect attempt will be made 
    // after a short delay. If a connection is re-established then the onConnect
    // handler will be invoked.
    //
    //  connected is true if a connection is established or false if disconnected.
    //  info contains human presentable information when available.
    //
    // If a reconnect is to take place with a different password or token then 
    // call disconnect(), followed by connectWithXyz().
    setConnectHandler(handler: (status: ConnectionStatus, info: string) => void): void


    // setEventHandler set the handler that receives all subscribed events, and other
    // message types, subscribed to by this client.
    //
    // This replaces any previously set event handler.
    //
    // See also 'Subscribe' to set the ThingIDs this client receives messages for.
    setEventHandler(handler: EventHandler): void

    // Subscribe adds a subscription for events from the given ThingID.
    //
    // This is for events only. Actions directed to this client are automatically passed
    // to this client's messageHandler.
    //
    // Subscriptions remain in effect when the connection with the messaging server is interrupted.
    //
    //  dThingID is the digital twin ID of the Thing to subscribe to.
    //	key is the type of event to subscribe to or "" for all events
    subscribe(dThingID: string, key:string): Promise<void>;

// Unsubscribe removes a previous event subscription.
// No more events or requests will be received after Unsubscribe.
    unsubscribe(dThingID: string): void;
}
