// ISubscription interface to underlying subscription mechanism
import type { IHiveKey } from "@hivelib/keys/IHiveKey";


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

// IHubTransport defines the interface of the message bus transport used by
// the hub client.
export interface IHubTransport {
    // addressTokens returns the address separator and wildcard tokens used by the transport
    // @result sep is the address separator. eg "." for nats, "/" for mqtt and redis
    // @result wc is the address wildcard. "*" for nats, "+" for mqtt
    // @result rem is the address remainder. "" for nats; "#" for mqtt
    addressTokens(): { sep: string, wc: string, rem: string };

    // ConnectWithPassword connects to the messaging server using password authentication.
    // @param loginID is the client's ID
    // @param password is created when registering the user with the auth service.
    connectWithPassword(password: string): Promise<void>;

    // ConnectWithToken connects to the messaging server using an authentication token
    // and pub/private keys provided when creating an instance of the hub client.
    // @param key is the key generated with createKey.
    // @param token is created by the auth service.
    connectWithToken(key: IHiveKey, token: string): Promise<void>;

    // CreateKeyPair returns a new key for authentication and signing.
    // @returns key contains the public/private key pair.
    createKeyPair(): IHiveKey;

    // Disconnect from the message bus.
    disconnect(): void;

    // PubEvent publishes event type messages and returns immediately. 
    // @param address to publish on
    // @param payload with serialized message to publish
    pubEvent(address: string, payload: string): Promise<void>;

    // PubRequest publishes an RPC request and waits for a response.
    // @param address to publish on
    // @param payload with serialized message to publish
    // @returns reply with serialized response message
    pubRequest(address: string, payload: string): Promise<string | boolean>;

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


    // Set the handler for incoming event-type messages.
    // Event type messages are those that do not contain a reply-to address and correlation data.
    //
    setEventHandler(handler: (addr: string, payload: string) => void): void

    // Set the handler for incoming request-response message.
    // The handler will be invoked if a message is received that contains a reply-to
    // address and a correlation ID. The result of the handler will be send back to
    // the sender. If an exception is thrown then an error will be returned to the sender.
    //
    // Support for request-response messages requires MQTT v5.
    //
    // The result of the handler will be sent as a reply.
    // This requires MQTT v5.
    setRequestHandler(handler: (addr: string, payload: string) => string): void

    // Subscribe adds a subscription for an event or request address.
    // Incoming messages are passed to the event handler or the request handler, depending on whether they
    // have a reply-to address. The event/request handler will handle the routing as this is application specific.
    // Subscriptions remain in effect when the connection with the messaging server is interrupted.
    //
    // The address MUST be constructed using the tokens provided by AddressTokens()
    subscribe(address: string): Promise<void>;

    // unsubscribe removes the address from the subscription list
    unsubscribe(address: string): void;
}
