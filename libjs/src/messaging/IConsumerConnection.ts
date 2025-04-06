import { RequestMessage, ResponseMessage} from "./Messages.ts";

export enum ConnectionStatus {
    Connected = "connected",
    Connecting = "connecting",
    ConnectFailed  = "connectFailed",
    Disconnected = "disconnected",
    // Unauthorized login name or password
    Unauthorized = "unauthorized"
}


export type ConnectionHandler = (status: ConnectionStatus)=>void;
export type RequestHandler = (msg: RequestMessage)=>ResponseMessage|null;
export type ResponseHandler = (resp: ResponseMessage)=>void;

// IConsumerConnection defines the interface of the consumer facing protocol binding.
export default interface IConsumerConnection {

    // ConnectWithPassword connects to the hub using password authentication.
    // @param password is created when registering the user with the auth service.
    // This returns an authentication token that can be used in refresh and connectWithToken.
    connectWithPassword(password: string): Promise<string>

    // ConnectWithToken connects to the messaging server using an authentication token
    // and pub/private keys provided when creating an instance of the hub client.
    // @param token is created by the auth service.
    connectWithToken(token: string): Promise<string>


    // Disconnect from the messaging server.
    disconnect(): void;

    // getStatus returns the current transport connection status
    // getStatus(): TransportStatus

    // invokeAction sends an action request and waits for a response or until timeout.
    // This is a simple helper that uses sendRequest(wot.OpInvokeAction, ...)
    //
    //	@param dThingID the digital twin ID for whom the action is intended
    //	@param name is the action or method name of the action to invoke
    //	@param input data to publish in native format as per TD
    //
    invokeAction(dThingID: string, name: string, input: any): Promise<ResponseMessage>

    // RefreshToken refreshes the authentication token
    // The resulting token can be used with 'ConnectWithJWT'
    refreshToken(): Promise<string>

    // Rpc sends a request message and waits for a response.
    //
    //  operation to request
    //	dThingID is the digital twin ID of the service providing the RPC method
    //	key is the ID of the RPC method as described in the service TD action affordance
    //	input is the value containing the input arguments sent
    //
    // This returns the data or throws an error if failed
    rpc(operation: string, dThingID: string, key: string, input: any): Promise<any>

    // sendRequest sends a request message and returns a response on completion or error.
    // if no response is received within a time period a timeout response is returned.
    sendRequest(msg: RequestMessage): Promise<ResponseMessage>

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


    // Set the progress handler
    // This replaces any previously set handler.
    setResponseHandler(handler: ResponseHandler):void

    // Subscribe adds a subscription for events from the given ThingID.
    //
    //  dThingID is the digital twin ID of the Thing to subscribe to. ""  for any
    //	name is the event name to subscribe to or "" for all events, "" for any
    subscribe(dThingID: string, name:string): void

// Unsubscribe removes a previous event subscription.
// No more events or requests will be received after Unsubscribe.
//     unsubscribe(dThingID: string, key: string): void;

    // writeProperty consumer requests a configuration change
    //  @param dThingID is the digitwin thingID is provided by the directory
    //	@param name ID of the property
    //	@param payload to publish in native format as per TD
    writeProperty(dThingID: string, name: string, payload: any):void
}
