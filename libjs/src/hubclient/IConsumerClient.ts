import type {IHiveKey} from "@keys/IHiveKey";
import {ThingMessage} from "@hivelib/things/ThingMessage";
import {RequestProgress} from "@hivelib/hubclient/RequestProgress";


export enum ConnectionStatus {
    Connected = "connected",
    Connecting = "connecting",
    ConnectFailed  = "connectFailed",
    Disconnected = "disconnected",
    // Unauthorized login name or password
    Unauthorized = "unauthorized"
}

export type ActionHandler = (msg:ThingMessage)=>RequestProgress;
export type EventHandler = (msg:ThingMessage)=>void;
export type PropertyHandler = (msg:ThingMessage)=>void;
export type ConnectionHandler = (status: ConnectionStatus)=>void;
export type ProgressHandler = (progress:RequestProgress)=>void;

// IAgentClient defines the interface of the hub agent transport.
export interface IConsumerClient  {

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

    // getStatus returns the current transport connection status
    // getStatus(): TransportStatus

    // invokeAction publishes an action request and returns as soon as the request is delivered
    // to the Hub inbox.
    //
    //	@param dThingID the digital twin ID for whom the action is intended
    //	@param name is the action or method name of the action to invoke
    //	@param payload to publish in native format as per TD
    //
    invokeAction(dThingID: string, name: string, requestID: string, payload: any): Promise<RequestProgress>;

    // RefreshToken refreshes the authentication token
    // The resulting token can be used with 'ConnectWithJWT'
    refreshToken(): Promise<string>

    // Rpc makes a RPC call using an action and waits for a delivery confirmation event.
    //
    // This is equivalent to use PubAction to send the request, use SetMessageHandler
    // to receive the delivery confirmation event and match the 'requestID' from the
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

    // set handler that is notified of changes in connection status and an error in
    // case of an  unintentional disconnect.
    // 
    // This handler is intended for updating presentation of the connection status.
    // Do not call connectXyz() in this handler, as a reconnect attempt will be made 
    // after a short delay. If a connection is re-established then the onConnect
    // handler will be invoked.
    //
    //  status contains the connection status, eg connected, disconnected
    //
    // If a reconnect is to take place with a different password or token then 
    // call disconnect(), followed by connectWithXyz().
    setConnectHandler(handler: (status: ConnectionStatus) => void): void

    // Set the action handler for incoming requests.
    // This replaces any previously set handler.
    setActionHandler(handler: ActionHandler):void

    // Set the event handler
    // This replaces any previously set handler.
    setEventHandler(handler: EventHandler):void

    // Set the property handler for incoming requests.
    // This replaces any previously set handler.
    setPropertyHandler(handler: PropertyHandler):void

    // Set the progress handler
    // This replaces any previously set handler.
    setProgressHandler(handler: ProgressHandler):void

    // Subscribe adds a subscription for events from the given ThingID.
    //
    //  dThingID is the digital twin ID of the Thing to subscribe to. ""  for any
    //	name is the event name to subscribe to or "" for all events, "" for any
    subscribe(dThingID: string, name:string): void;

// Unsubscribe removes a previous event subscription.
// No more events or requests will be received after Unsubscribe.
//     unsubscribe(dThingID: string, key: string): void;

    // writeProperty consumer requests a configuration change
    //  @param dThingID is the digitwin thingID is provided by the directory
    //	@param name ID of the property
    //	@param payload to publish in native format as per TD
    writeProperty(dThingID: string, name: string, payload: any):void;
}
