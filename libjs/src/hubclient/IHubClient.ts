// ISubscription interface to underlying subscription mechanism
import type {IHiveKey} from "@keys/IHiveKey";
import {ThingMessage} from "@hivelib/things/ThingMessage";
import {TD} from "@hivelib/things/TD";


export enum ConnectionStatus {
    Connected = "connected",
    Connecting = "connecting",
    ConnectFailed  = "connectFailed",
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

    // DeliveryToAgent the request is delivered to the Thing's agent by the inbox.
    // The agent is expected to send a delivery update.
    DeliveryToAgent = "agent",

    // DeliveryWaiting optional step where the agent is waiting to apply it to
    // the Thing, for example when the device is asleep.
    // This status is sent by the agent.
    // An additional progress update from the agent can be expected.
    DeliveryWaiting = "waiting",

    // DeliveryApplied is a step where the request has been applied to the Thing by the
    // agent.
    // If the device is busy processing then a confirmation 'DeliverySuccess'
    // is sent when the device confirms execution.
    // If the device fails to execute the request an error is included and the
    // workflow has ended.
    DeliveryApplied = "applied",

    // DeliverySuccess the request has been applied by the agent and the device
    // has executed the request successfully.
    // Obtaining a result is not always possible for example when a device is asleep.
    // In that case 'applied' is the last known status sent.
    //
    // Issue: if a node asynchronously notifies that an update changed a value without
    // a link to the request, then how to update the status of the request to completed?
    // Should the inbox that sees a value event for an action that is in state
    // 'applied' change it to 'result'?
    //
    DeliveryCompleted = "completed",

    // DeliveryFailed the request could not be delivered to the agent or the agent can
    // not deliver the request to the Thing. This ends the delivery process.
    // The error field contains the error message describing the failure.
    DeliveryFailed = "failed",

}

// DeliveryStatus holds the progress of action request delivery
export class DeliveryStatus extends Object{
    // Request ID
    messageID: string = ""
    // Updated delivery status
    progress: DeliveryProgress|undefined
    // Error in case delivery status has ended without completion.
    error?: string =""
    // Reply in case delivery status is completed
    // FIXME: change reply to any
    reply?: string =""

    completed(msg: ThingMessage, err?: Error) {
        this.messageID = msg.messageID
        this.progress = DeliveryProgress.DeliveryCompleted
        if (err) {
            this.error = err.name + ": " + err.message
        }
    }
    // set status update to applied. A final status update is expected
    applied(msg: ThingMessage) {
        this.messageID = msg.messageID
        this.progress = DeliveryProgress.DeliveryApplied
        this.error = undefined
    }
    failed(msg: ThingMessage, err: Error|string|undefined) {
        this.messageID = msg.messageID
        this.progress = DeliveryProgress.DeliveryFailed
        if (err instanceof Error) {
            this.error = err.name + ": " + err.message
        } else {
            this.error = err
        }
    }
}

export type EventHandler = (msg:ThingMessage)=>void;

export type MessageHandler = (msg:ThingMessage)=>DeliveryStatus;

// IHubClient defines the interface of the hub transport client.
export interface IHubClient {

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
    //	@param payload to publish in native format as per TD
    //
    pubAction(dThingID: string, key: string, payload: any): Promise<DeliveryStatus>;

    // getStatus returns the current transport connection status
    // getStatus(): HubTransportStatus

    // PubEvent publishes an event style message without a response.
    // It returns as soon as delivery to the hub is confirmed.
    //
    // Events are published by agents using their native ID, not the digital twin ID.
    // The Hub outbox broadcasts this event using the digital twin ID.
    //
    //	@param thingID native thingID as provided by the agent
    //	@param key ID of the event
    //	@param payload to publish in native format as per TD
    //
    // This throws an error if the event cannot not be delivered to the hub
    pubEvent(thingID: string, key:string, payload: any): Promise<DeliveryStatus>;

    // pubProperty publishes a configuration change request for a property
    //  @param dThingID is the digitwin thingID is provided by the directory
    //	@param key ID of the property
    //	@param payload to publish in native format as per TD
    pubProperty(dThingID: string, key: string, payload: any): Promise<DeliveryStatus>;

    // PubProps publishes a property values event.
    // It returns as soon as delivery to the hub is confirmed.
    // This is intended for agents, not for consumers.
    //
    // @param thingID is the native thingID of the device (not including the digital twin ID)
    // @param props is the property key-value map to publish where value is their native format
    //
    // This throws an error if the event cannot not be delivered to the hub
    pubProps(thingID: string, props: {[key:string]:any}): Promise<DeliveryStatus>;

    // PubTD publishes an TD document event.
    // It returns as soon as delivery to the hub is confirmed.
    // This is intended for agents, not for consumers.
    //
    // @param td is the Thing Description document describing the Thing
    pubTD(td: TD): Promise<DeliveryStatus>

    // RefreshToken refreshes the authentication token
    // The resulting token can be used with 'ConnectWithJWT'
    refreshToken(): Promise<string>

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
    rpc(dThingID: string, key: string, args: any): Promise<any>

    // SendDeliveryUpdate sends a delivery progress update event to the hub.
    // The hub's inbox will update the status of the action and notify the original sender.
    //
    // Intended for agents that have processed an incoming action request asynchronously
    // and need to send an update on further progress.
    sendDeliveryUpdate(stat: DeliveryStatus):void


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


    // Set the handler for incoming requests.
    // This replaces any previously set handler.
    setMessageHandler(cb: MessageHandler):void

    // Subscribe adds a subscription for events from the given ThingID.
    //
    // This is for events only. Actions directed to this client are automatically passed
    // to this client's messageHandler.
    //
    // Subscriptions remain in effect when the connection with the messaging server is interrupted.
    //
    //  dThingID is the digital twin ID of the Thing to subscribe to. "" or "+" for any
    //	key is the type of event to subscribe to or "" for all events, "" or "+" for any
    subscribe(dThingID: string, key:string): Promise<void>;

// Unsubscribe removes a previous event subscription.
// No more events or requests will be received after Unsubscribe.
    unsubscribe(dThingID: string): void;
}
