# HiveOT Transport Protocol Bindings

The HiveOT transports package has two main objectives:
1. Support any protocol to exchange messages between consumers and agents.
2. Standardize the messaging for consumers and agents regardless the transport used.

Consumers are end-users of IoT devices and services. WoT specifies protocol bindings to describe how consumers interact with Things. Unfortunately these protocol bindings each differ in the message envelopes used. HiveOT bindings therefore maps between the standardized hiveot message format and that of the protocol binding, while remaining compatible with the WoT specification where possible. 

Agents are services that expose a Thing to consumers. HiveOT uses an agent model where agents are clients of a Hub, just like consumers. WoT only addresses consumer-server interaction, so agent-server messaging is not specified. Since WoT isn't providing any specifications for this, HiveOT makes a best effort to use WoT concepts and vocabulary as far as they exists.

HiveOT bindings supports the three hiveOT standardized message envelopes over the http, sse, ws and mqtt transports: These envelopes match the message flow for requests, responses and notifications, and are passed as-is over the underlying transport. This is used as the fallback in case forms are not available. This effectively act as a WoT application protocol that provides easy message exchange regardless the underlying transport. 


## Messaging on HiveOT

Hiveot APIs uses standardized message envelopes to provided a simple and consistent API regardless the protocol binding in use. Protocol bindings effectively map between the protocol specific messages and the hiveot standardized messages. The hiveot messages become the application level protocol that is missing in the WoT standard. It is the hope of HiveOT that this or a similar standard will be adopted by the WoT to provide a unified messaging interface for all protocol bindings.

This approach builds on top of the WoT definition for 'operations' and adds operations for use by agents, supporting agents as clients of a hub or gateway. 

### Notification Message

Notifications serve to notify subscribers of a change as identified by the operation, thingID and affordance name. Notifications are intended for any number of subscribers (or observers) and do not receive a response.

All notifications use the same message envelope as implemented in NotificationMessage struct (golang, JS, Python). Protocol bindings can use this envelope directly or map from their protocol equivalent to this message format.

The following operations are considered notifications:
* property:  Update of a property value, sent by a Thing agent to observers of a property.
* event:  Notification of event to subscribers.
* td: Update of a TD by the directory (hiveot extension)

| name              | data type | description                                                                                                                                        | required  |
|-------------------|-----------|----------------------------------------------------------------------------------------------------------------------------------------------------|-----------|
| type              | string    | "notification". Identifies the message as a notification message                                                                                   | mandatory |
| operation         | string    | Identification of the notification                                                                                                                 | mandatory |
| data              | any       | notification data as specified by the operation                                                                                                    | optional  |
| correlationID | string | optional correlation with the request that caused the notification, such as subscriptions or message streams                                       | optional  |
| created           | string    | Timestamp the notification was created                                                                                                             | optional  |
| thingID           | string    | ID of the thing the notification applies to                                                                                                        | optional  |
| name              | string    | Name of the affordance the notification applies to, if applicable. The type of affordance (event, action, property) is determined by the operation | optional  |
| messageID         | string    | Unique identification of the message.                                                                                                              | optional  |


### Request Message

The purpose of the request message is for a client to send a request for an operation on a thing.  Thing agents are required to send a response when a request is received.

The following operations are considered to be requests:
* invokeaction  [WoT]
* subscribe, unsubscribe [WoT]
* observe, unobserve [WoT]
* readproperty, readallproperties [WoT]
* queryaction, queryallactions [WoT]
* readevent, readallevents  (of a Thing)  [HiveOT extension]
* readtd, readalltds  (of a directory or thing) [HiveOT extension]

The request message defines the following fields:

| name      | data type | description                                                                                                                                                             | required  |
|-----------|-----------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------|
| type      | string| "request". Identifies the message as a request message                                                                                                                  | mandatory |
| operation | string| Describes the request to perform                                                                                                                                        | mandatory |
| thingID   | string| ID of the thing the request applies to                                                                                                                                  | optional  |
| name      | string| Name of the affordance the request applies to if applicable. The type of affordance (event, action, property) is determined by the operation                            | optional  |
| input     | any   | Input data of the request as described by the operation. invokeaction refers to the action affordance while other operations define the input as part of the operation  | optional  |
| correlationID | string| Unique identifier of the request. This must be included in the response. If no correlationID is provided then the request will still be handled by no response is returned. | optional  |
| senderID  | string| Authenticated sender of the request.                                                                                                                                    | optional  |
| messageID | string| Unique identification of the message.                                                                                                                                   | optional  |


### Response Message

Responses serve to notify a single client of the result of a request.

Response message payload is determined by the request operation. Therefore the request operation is included in the response:

| name      | data type   | description                                                                                                                                                                                  | required  |
|-----------|--------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------|
| type      | string | "response". Identifies the message as a response message                                                                                                                                     | mandatory |
| correlationID | string | identifies the request this is a response to.                                                                                                                                                | required  |
| status    | string | Status of the request processing: "pending", "running", "completed" or "failed"                                                                                                              | required  |
| output    | any    | Result of processing the request if status is "completed" as defined in the dataschema of the action or operation. If status is "failed" then this can contain additional error information. | optional  |
| error     | string | Error title if status is "failed".                                                                                                                                                           | optional  |
| received  | string | Timestamp the request was received by the Thing (or its agent)                                                                                                                               | optional  |
| updated   | string | Timestamp the status was updated                                                                                                                                                             | optional  |
| operation | string | The request operation this is a response to, to aid in debugging.                                                                                                                            | optional  |
| thingID   | string | ID of the thing the request applies to aid in debugging.                                                                                                                                     | optional  |
| name      | string | Name of the affordance the request applies to if applicable. The type of affordance (event, action, property) is determined by the operation                                                 | optional  |
| messageID | string | Unique identification of the message.                                                                                                                                                        | optional  |

### Behavior

Any client can publish a request. They can choose to wait for a response message or handle the response asynchronously. This is dependent on the client implementation.  

The server can be an agent for a thing or a hub or gateway. 

Hub or gateways will forward requests to the Thing agent. If the agent is not reachable at this point they return the error response or a pending response if there is support for queuing requests.

Thing agents will process the request. They SHOULD send a response with one of the statuses: pending, running, completed or failed. If a request is received without a request ID then no response is sent.

If a 'running' response is send, agents MUST also send a completed or failed response once the operation has finished.

If a hub or gateway is used then the response is received by the hub/gateway, which in turn forwards it to the client that sent the request. If the client is no longer reachable then the response can be queued or discarded, depending on the capabilities of the hub or gateway.

Clients will receive a response message containing the original correlationID, the payload an any error information. Client implementations can choose to wait for a response (with timeout) or handle it as a separate callback.

Agents can also sent notifications to subscribers. The notification message includes an operation that identifies the type of notification. A correlationID is optional to link the notification to a subscription request. A hub or gateway can implement their own subscription mechanism for consumers and ask agents to send all notifications.



## Implementation

In HiveOT both consumers and agents connect to a Hub for exchanging information. Therefore Thing agents are implemented as clients. Transport servers receive requests. It is possible to use the transport server to implement a Thing and act as an agent, but this is not the intent of hiveot. 

For each protocol binding this library provides 3 components:
* Server transports that receive connections and handle messages from both agents and consumer clients. This library is named XyzServerTransport, where Xyz is the protocol name. (eg HttpServerTransport)
* Consumer transports that can submit requests and receive responses. This library is named XyzConsumerTransport.
* Agent transports that extend the consumer clients with receive requests and sending responses. This library is named XyzAgentTransport.

Each protocol binding implements the message formats as defined by their specification. These bindings map the protocol specific messages into the universal hiveot message format so applications don't have to consider the protocol binding used.

### Consumer Transports
Consumer transports implement the IConsumerConnection interface containing these methods:

1. SendRequest - send a request for operation to a Thing agent or service and optionally wait for a response or confirmation. Intended for both consumers and agents. The InvokeAction operation for example is sent using SendRequest, as are ReadProperty and such. The client must include a request-id that an asynchronous response can be addressed to.
2. Subscribe/Observe - register to receive events or property updates
3. SetNotificationHandler - receive a property, event, action status, and td notification message.

### Agent Transports
Agent transports implement the IAgentConnection interface containing these methods:

1. SetRequestHandler - receive an action, property write, readproperty, query action requests message.
2. SendNotification - send an operation to subscribers or observers. No response is expected. Intended for Thing agents or services.
3. SendResponse - send a response to a request. This requires the request-id to respond to. The 'operation' describes the type of response and differs from the request operation. 

### Server Transports

The server transport listens for incoming connections from both agents and consumers. They convert incoming messages to a standard request, response or notification messages and pass it on to the corresponding handler. In case of the HiveOT Hub this is the digital twin router.

Server transports aim to be WoT compliant to support use by 3rd party WoT consumers.

These implement the ITransportServer interface containing these methods:
1. AddTDForms - add forms to the given TD for supported operations using this transport. 
2. GetForm - return the form for a specific operation. Primarily intended for testing.
3. GetConnectURL - provide the URL used to connect to this server
4. SendNotification - broadcast an event or property change notification to subscribers
6. SetConnectionHandler - incoming connections are reported as an instance of the IServerConnection interface for sending messages to the remote client:
   * SendNotification - send a notification to the connected consumer client
   * SendRequest - send a request to the connected agent client. An asynchronous response message is expected.
   * SendResponse - send a response to a request to the connected consumer client.

#### Authentication

TransportServers can provide facility to authenticate the client and MUST support the facility to verify identity before processing messages. Once a message is passed to the message handler, the client is considered to be authenticated. The only exception is the login request which is handled out of bounds, depending on the transport protocol. 

### Discovery

Before a consumer or agent can connect to the server, the server must be discovered. 
Transport servers use DNS-SD to publish their discovery information.

Compliance with [WoT Discovery](https://www.w3.org/TR/wot-discovery/) is planned.

This transport library includes a ConnectToHub method that discovers the server and establishes a connection using the discovered protocol. If multiple protocols are supported the most efficient protocol is used. Currently this is a fixed order: Websockets, MQTT, HTTP/SSE-SC. Other protocols will be added in the future.  

# Supported Transport Protocols

## HTTP-Basic Transport 

HTTP-Basic is used as a base for HTTP/SSE-SC and HTTP/WSS. As it only supports synchronous request-responses from client to server, it cannot be used as TransportAgent.

Limitations:
1. There is no confirmation of an action on a remote Thing.
2. There is no output of an action on a remote Thing.
3. Subscription to events is not supported
4. Observing properties is not support.

HttpConsumerTransport operations:
* SendRequest
  * Operation Invoke Action
    * Receive synchronous results only 
    * Results from remote Thing agents are not received 
  * Operation Write Property
    * Receive synchronous results only
	* Results from remote Thing agentsare not received
  * Operation Read/Query property/action/events 
    * Receive synchronous results only
    * HiveOT returns results from the digital twin synchronously
* Ping 

HttpAgentTransport operations (extensions to WoT) :
* SendNotification 
  * property update
  * events
  * actionstatus 
* SendResponse (to requests) - not supported

HttpServerTransport handles:
* receive requests - forward to handler
* receive notifications - forward to handler
* ping - synchronous response

Forms: 
Forms use a generalized request path for all operations in the format:
> /http/{operation}[/{thingID}[/{name}]]

Where {thingID} and {name} are optional URI variables and only used when applicable.

The method used depends on the operation as described in the HTTP-Basic specification.

Examples:

To return the TD of a thing:
> GET https://server/http/readtd/{thingID}

To read a Thing property value:
> GET https://server/http/getproperty/{thingID}/{name}

These operations are described in a TD top level form as per [TD-1.1 specification](https://www.w3.org/TR/wot-thing-description11/#form):
```json
{
  "forms": [
    {
      "op": "readproperty",
      "href": "/http/readproperty/dtw:agentID:thingID/propertyName",
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
2. This is only intended for use by consumers. Remote agents cannot use this to receive requests as this is not supported in the WoT standard.
3. A separate connection is needed for subscribing to each thing. Subscribing to 10 Things for example requires 10 connections. This is only scalable when using HTTP/2. 

Example: A form that could subscribe to all events of a Thing looks like:
```json
{
  "forms": [
    {
      "op": ["subscribeallevents","unsubscribeallevents"],
      "href": "/sse/subscribeallevents/dtw:agent1:thing1",
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
2. A non-wot message envelope is used to include extra information such as operation, thingID, affordance name, senderID and correlationID.
3. HTTP headers are used to define a connection ID ["cid"] to correlate http requests with the SSE connection that sends the replies.
4. HTTP headers are used to include a correlationID ["correlationID"] in requests and responses. (this name is subject to change)

Example:

Forms to connect, subscribe and unsubscribe events of a Thing could look like:
```json
{
  "forms": [
    {
      "op": "subscribeallevents",
      "href": "/ssesc/subscribeallevents/dtw:agent1:thing1",
      "htv:methodName": "GET",
      "subprotocol": "ssesc",
      "headers": ["cid", "correlationID"]
    },
    {
      "op": "observerproperty",
      "href": "/ssesc/observerproperty/dtw:agent1:thing1/property1",
      "htv:methodName": "GET",
      "subprotocol": "ssesc",
      "headers": ["cid"]
    }
  ]
}
```

To subscribe to events, post to the subscribe endpoint with the thingID. This returns with a 200 code on success or 401 when the consumer is not allowed subscriptions to this Thing.

Include a 'cid' header field containing a connection-id. This is required for linking the subscription to the SSE connection that receives the events or property updates. .

Once a request is received, an agent has to send a response. When the agent is another client, as in hiveot, the response has to be linked to the request. When the agent sends a response message it must therefore include the correlationID provided with the request. The response operation field must be empty unless it contains a known response operation, such as actionstatus. In hiveot this is used to send a response to the consumer that send the request.



### HTTP+WebSocket Transport

This transports uses HTTP to establish a connection and authenticate. All further messages take place using websocket messages.
The WoT draft proposal defines message envelopes for each operation, including requests, subscriptions and responses.

Limitations:
1. Agent push is not a WoT standard. There are no operations defined for publishing a TD, events and updating properties. Hiveot needs to add a context extension for these operations.
   * TBD, can the hub query the TD over the websocket connection? 

Discovery publishes the http address, port and websocket address.

A form to subscribe to all events of a Thing could look like:
```json
{
  "forms": [
    {
      "op": "subscribeallevents",
      "href": "/wss/subscribeallevents/dtw:agent1:thing1",
      "subprotocol": "wss"
    },
    {
      "op": "observerproperty",
      "href": "/wss/observerproperty/dtw:agent1:thing1/property1",
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
