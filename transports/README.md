# HiveOT Transport Protocol Bindings

Transport bindings consists of a server and a client component for each protocol.
Servers are implemented in golang and embedded in the runtime.
Clients can be implemented in any language. A golang implementation is included. 

A key objective is that consumers and Thing agents can each use their own preferred protocol. Things (agents) and consumers only need to implement a single protocol to work with each other. The HiveOT Hub converts the messages between protocols using the transports described below.

Hiveot defines standard interfaces for transport bindings so they can be used interchangeably.   

Transport clients must implement the following methods for use by the application. See also IClientConnection:
1. SendNotification - send a message to subscribers or observers. No response is expected. Intended for Thing agents or services.
2. SendRequest - send a request to a Thing agent or service and wait for a response or confirmation. Intended for both consumers and agents. The InvokeAction operation for example is sent using SendRequest.
3. SendResponse - send a response to a request. Intended for use by agents to reply. For example, agents use SendResponse to reply to a request.
4. Subscribe/Observe - register to receive events or property updates

Fundamentally there are three types of transports.
- Unidirectional transports that can only send messages one way on a connection and receive no confirmation or response. For example SSE. 
- Unidirectional transports that can only send messages one way on a connection and receive confirmation and a response. For example HTTP. 
- Bidirectional transports where messages can pass both ways between client and server. For example: Websocket, MQTT, gRPC, CoAP (?). These need additional protocol support to correlation the response with a request.

HiveOT transport protocol bindings support HTTP standand-alone and in combination with SSE. Websockets are supported using the same HTTP connection. All protocols are encrypted, so HTTPS is used for HTTP, SSE and websockets.

In HiveOT both consumers and agents are remote clients. This adds the requirement that agents must use bi-directional transports. In theory agents  can also use polling over HTTP, creating a semi-bi-directional transport.


## HTTP Transport

Pure HTTP is uni-directional. The server cannot push messages to the client, so the client is required to poll for updates by sending a request.

1. Send notifications and request messages
2. Confirms thats messages have been sent successfully
3. Returns a response only when immediately available
4. Does NOT support receiving requests from the server (agents)
5. Does NOT support subscribe and observe operations 

For the above reason, HTTP-only clients are only used by 3rd party consumers. It is the most basic and limited form of interoperability.
HTTP can be combined with a sub-protocol to add a return channel, as described below.

### HTTP Basic Transport

HTTP supports including results in the HTTP response body. It can therefore include a reply to requests as long as they are immediately available. 

Limitations:
1. There is no confirmation of an action on a remote Thing.
2. There is no output of an action on a remote Thing. 
3. Subscription to events is not supported
4. Observing properties is not support.

* Note that reading properties and querying actions does return a result as this is an interaction with the digital twin, which returns results immediately.
* A possible workaround to the first two limitations is to have the server wait for a response when forwarding requests to remote agents. This has the problem that it blocks the http connection until a response is received.

Future: 
1. Support 'full compatibility mode' by having the server wait for a result of actions on remote Things. Enable/Disable this feature in the configuration. The downsides are that it can block the client connection until a result is received and it can use up a lot of resources.

Examples:

To return the TD of a thing:
> GET https://server/digitwin/td/{thingID}


To read a Thing property value:
> GET https://server/digitwin/properties/{thingID}/{name}

These operations are described in a TD top level form as per [TD-1.1 specification](https://www.w3.org/TR/wot-thing-description11/#form):
```json
{
  "forms": [
    {
      "op": "readproperty",
      "href": "/digitwin/properties/dtw:agent1:thing1/{name}",
	  "htv:methodName": "GET"
    }
  ]
}
```

### HTTP+SSE Transport

This transports adds a return channel to HTTP using the SSE sub-protocol. The WoT specification states that the connection path contains the resource to subscribe or observe. Additional subscriptions require a new connection. 

Discovery publishes the http address and port, and the path to retrieve the Thing directory.

Limitations: 
1. This is currently not fully implemented 
2. Remote agents cannot use this to receive requests as this is not supported in the WoT standard.
3Only partially interoperable with http-only clients. Only immediate results from the digital twins are returned in the http body. Results from actions are send async. Same problem as http-only servers. 
4. A separate connection is needed for subscribing to each thing. Subscribing to 10 Things for example requires 10 connections. This is only scalable when using HTTP/2. 

Additional complications:
1: how to return synchronous vs async results? 
	* Option 1: immediate results are included in the http response. Async results are send separately as SSE messages.
		* Pro: This is partly compatible with regular 3rd party http clients. 
		* Con: The SSE binding client somehow needs to deal with results from two sources (http response and async response)
		* Con: async results from actions on remote Things are not available to regular http clients.
	* Option 2: always return results async via SSE
		* Pro: easier to implement and handle
		* Con: not compatible with 3rd party http consumers. 

TODO: implement connecting as sse 

Example: A form that could subscribe to all events of a Thing looks like:
```json
{
  "forms": [
    {
      "op": ["subscribeallevents","unsubscribeallevents"],
      "href": "/sse/digitwin/events/dtw:agent1:thing1",
	  "htv:methodName": "GET",
      "subprotocol": "sse"
    }
  ]
}
```


### HTTP+SSE_SC Transport

This transport shares a single SSE return channel for all messages, subscriptions and observations from the connected client. It uses the SSE event ID to pass operation, ThingID and affordance name, which allows for identifying messages for multiple things, affordances and operations.
HTTP methods are used to (un)subscribe events and (un)observe properties. These are defined in the Forms.

Limitations:
1. This is not a WoT SSE subprotocol specification (maybe it should be)
2. Server push over SSE is not a WoT standard. There are no operations defined so hiveot needs to add a context extension.
3. Only partially interoperable with http-only clients. Only immediate results from the digital twins are returned in the http body. Results from actions are send async. Same problem as http-only servers.

Discovery publishes the http address and port, and the path to retrieve the Thing directory. 

Example:

Forms to connect, subscribe and unsubscribe events of a Thing could look like:
```json
{
  "forms": [
    {
      "op": "sse:connect",
      "href": "/ssesc",
      "htv:methodName": "GET",
      "subprotocol": "ssesc",
      "headers": ["cid"]
    },
    {
      "op": "subscribeallevents",
      "href": "/ssesc/digitwin/subscribe/dtw:agent1:thing1",
      "htv:methodName": "POST",
      "subprotocol": "ssesc",
      "headers": ["cid", "requestID"]
    },
    {
      "op": "unsubscribeallevents",
      "href": "/ssesc/digitwin/unsubscribe/dtw:agent1:thing1",
      "htv:methodName": "POST",
      "subprotocol": "ssesc",
      "headers": ["cid"]
    }
  ]
}
```

This introduces a 'connect' operation for establishing the sse connection. This operation is only needed once for all subscriptions. This establishes an SSE connection as per RFC or returns 401 when the consumer does not provide proper credentials connecting to the sse endpoint.

To subscribe to events, post to the subscribe endpoint with the thingID. This returns with a 200 code on success or 401 when the consumer is not allowed subscriptions to this Thing.

Optionally supply a 'cid' header field containing a connection-id. This is required for linking a subscription to a connection from the same client, as is the case in multiple browser tabs where each tab has its own connection using the authentication token in a shared cookie.

### HTTP+WebSocket Transport

This transports uses HTTP to establish a connection. All further messages take place using websocket messages.
The WoT draft proposal defines message envelopes for each operation, including requests, subscriptions and responses.

Limitations:
1. Server push is not a WoT standard. There are no operations defined for publishing a TD, events and updating properties. Hiveot needs to add a context extension for these operations.
   * TBD, can the hub query the TD over the websocket connection? 

Discovery publishes the http address, port and websocket address.

A form to subscribe to all events of a Thing could look like:
```json
{
  "forms": [
    {
      "op": "subscribeallevents",
      "href": "/wss/digitwin/subscribe/dtw:agent1:thing1",
      "subprotocol": "wss"
    },
    {
      "op": "unsubscribeallevents",
      "href": "/wss/digitwin/unsubscribe/dtw:agent1:thing1",
      "subprotocol": "wss"
    }
  ]
}
```

This follows the specification from the 'strawman proposal' https://docs.google.com/document/d/1KWv-aQfMgsqBFg0v4rVqzcVvzzisC7y4X4CMUYGc8rE by Ben Francis. 

## MQTT Transport Server (todo)

The HiveOT Hub runs a MQTT broker. Consumers and agents connect to the broker MQTT over TCP or Websocket. 

Discovery publishes the MQTT address, TPC and Websocket ports.

This is not intended for connecting to 3rd party brokers. This would require a separate protocol binding

## MQTT Transport Bridge (todo)

Intended to interoperate with 3rd party WoT devices using a 3rd party MQTT.


## CoAP Transport (todo)

this is under investigation

## Discovery (todo)

Discovery provides a means for consumers and agents to locate the Hub and query the directory of Things. The Thing TD include Forms for interacting with the Thing.

The Hub uses DNS-LD to publish its address on the local network.

## Authentication

The Hub maintains a list of clients, their credentials and roles. In order to authenticate, the client ID and credentials must be known.

All transport bindings support the use of authentication tokens to authenticate a connection. The HTTP binding provides a login method to obtain and refresh a token. 

Authentication using client certificates are planned for the future.

OAuth2 support is planned for the future. 
Radius support is planned for the future

## Connecting to 3rd Party WoT devices

The Hub is based on the concept that IoT devices (agents) connect to it. The WoT specification is designed around the idea that consumers connect directly to THing agents however.

For consumers there is no difference between connecting to a Thing directly or to the HiveOT Hub. The only requirement is that they know that the Hub is a gateway for multiple Things, so they need to read the directory after connecting. From there on, Forms in the TD will provide the Hub endpoint to connect to. Any interaction is WoT compatible, depending on the protocol used. 

For stand-alone Things the situation is different as they expect consumers to connect to them. To interoperate, the Hub implements a WoT binding that searches for WoT compatible devices and connect to them using configured credentials. The Hub acts as the consumer for these devices. (This binding is not yet implemented.)
