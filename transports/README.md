# HiveOT Transport Bindings

## Introduction

The HiveOT transports package has the following objectives:
1. Support any protocol to exchange messages between consumers and agents.
2. Standardize the messaging for consumers and agents regardless the underlying transport used.

In HiveOT the Hub acts as the Thing agent for digital twins. Consumers connect to the Hub and can use all the Things available on the Hub as if it was the Thing itself. 

The hub itself is a consumer of Things which are used to create a digital twin on the Hub. The main difference between hub and agent is that agents connect to the Hub instead of the other way around. The messaging however is still that of a consumer (hub) talking to an agent. In theory the connection can also be made by the hub, after which the exchange continues in the same way.

This transports package is designed such that it can also be used stand-alone with requiring the Hub. The protocol clients and servers are used by consumer, agents and the hub.

HiveOT provides a messaging layer on top of the WoT protocol bindings. This layer offers a unified way of communicating between consumer and Thing regardless the underlying transport protocol. In a addition, a set of direct transports are provided that simply transport the messages as-is instead of converting them to the WoT protocol binding format. These direct transports work over http/sse, wss, mqtt and other and use the request and response envelopes.

This package has the following folder structure:
```
   root (transports)
     - clients      WoT protocol bindings for communicating with a WoT compatible server
     - servers      WoT protocol server for communicating with consumers
     - connections  Connection and subscription manager for use by servers
     - messaging    Application messaging layer built on top of clients and servers
     - tputils      Transport protocol helper classes and methods
     - tests        Test cases that operate on the messaging layer for testing protocol bindings. 
```


## Messaging on HiveOT

The messaging layer is a thin application layer built on top of the transport protocols. It provides a simple WoT friendly API to consumers and Thing agents for building IoT applications and Things.

The purpose of this messaging layer is to enable WoT applications to work over any support transport protocol.

HiveOT messaging uses a simple request-response message envelope. Clients send requests for operations and receive responses for these operations. It supports both direct and asynchronous transports. 

This approach builds on top of the WoT definition for 'operations' and adds operations for use by agents, supporting agents as clients of a hub or gateway. 

### Request Message

The purpose of the request message is for a client to send an asynchronous request for an operation on a thing. Thing agents implemented using these messages are required to send a response when a request is received.

The following operations are considered to be requests:
* invokeaction  [WoT]
* subscribe, unsubscribe [WoT]
* observe, unobserve [WoT]
* readproperty, readallproperties [WoT]
* queryaction, queryallactions [WoT]
* readevent, readallevents  (of a Thing)  [HiveOT extension]
* readtd, readalltds  (of a directory or thing) [HiveOT extension]
* subscribe, unsubscribe
* observe, unobserve

The request message defines the following fields:

| name          | data type | description                                                  | required  |
|---------------|-----------|--------------------------------------------------------------|-----------|
| type          | string    | "request". Identifies the message as a request message       | Required  |
| operation     | string    | Describes the request to perform                             | Required  |
| messageID     | string    | Unique identification of the message.                        | Required  |
| correlationID | string    | Unique identifier of the request.                            | Optional  |
| thingID       | string    | ID of the thing the request applies to                       | Optional  |
| name          | string    | Name of the affordance the request applies to if applicable. | Optional  |
| input         | any       | Input data of the request as described by the operation.     | Optional  |
| senderID      | string    | Authenticated sender of the request.                         | Optional  |


### Response Message

Responses serve to provide a client the progress or result of a request.

Response message payload is determined by the request operation and the response status. Therefore the request operation is included in the response:

| name          | data type | description                                                                     | required  |
|---------------|-----------|---------------------------------------------------------------------------------|-----------|
| type          | string    | "response". Identifies the message as a response message                        | required  |
| operation     | string    | The request operation this is a response to, to aid in debugging.               | optional  |
| messageID     | string    | Unique identification of the message.                                           | required  |
| correlationID | string    | identifies the request this is a response to.                                   | required  |
| thingID       | string    | ID of the thing the request applies to aid in debugging.                        | optional  |
| name          | string    | Name of the affordance the request applies.                                     | optional  |
| output        | any       | Result of processing the request when status is "completed".                    | optional  |
| status        | string    | Status of the request processing: "pending", "running", "completed" or "failed" | required  |
| error         | string    | Error title if status is "failed".                                              | optional  |
| requested     | string    | Timestamp the request was received by the Thing (or its agent)                  | optional  |
| updated       | string    | Timestamp the status was updated                                                | optional  |

### Behavior

Any client can send a request. They can choose to wait for a response message or handle the response asynchronously. Sending the request involves an application protocol client and a transport protocol client. The HiveOT application protocol is identical regardless the underlying transport protocol used. The transport protocol is one of the client transports like http/sse-sc, websocket mqtt, etc. More on this below.

HiveOT uses a simple application protocol built on top of transport protocols. Clients only need to be aware of this protocol, which will work with any of the supported transport protocols.  

The application protocol consists of only 2 messages. RequestMessage and ResponseMessage.

Requests are handled by the protocol server, which is run by the HiveOT Hub, or a Thing agent. The Hub serves digital twin requests while Thing agents server device requests.

While this package is intended for connecting to the HiveOT Hub, it can also be used connecting between consumers and 3rd party Thing agents, including services. Use of the Hub is not a requirement for using this package.

When used with the Hub, these protocol clients support a reversal of connection, where they connect to the Hub instead of consumers connecting to the agent. Once the connection is established the message flow is the same between consumer and hub, and between hub and agent. 

Consumers connect to the protocol server using connection information described in TD forms. Agents connect to the Hub as a client using out-of-band token authentication. 

Hub or gateways will forward requests to the Thing agent. If the agent is not reachable at this point they return the error response or a pending response if there is support for queuing requests.

Thing agents will process the request. They SHOULD send a response with one of the statuses: pending, running, completed or failed. If a request is received without a request ID then no response is sent.

If a 'running' response is sent, agents MUST also send a completed or failed response once the operation has finished.

If a hub or gateway is used then the response is received by the hub/gateway, which in turn updates the digital twin and passes the digital twin response to the client that sent the request. If the client is no longer reachable then the response can be queued or discarded, depending on the capabilities of the hub or gateway.

Clients will receive a response message containing the correlationID provided in the request, the payload, and any error information. Client implementations can choose to wait for a response (with timeout) or handle it as a separate callback.

Agents publish notifications to subscribers. This is a 1-many relationship. The notification message includes an operation that identifies the type of notification. A correlationID is optional to link the notification to a subscription request. A hub or gateway can implement their own subscription mechanism for consumers and ask agents to send all notifications.


## Design

There are 3 layers involved in exchanging messages:
1: messaging: sending a request and receiving a response using the underlying protocol transport.
2: mapping: map between request and response messages and the underlying transport protocol binding messages.
3: transport: establishing a connection to send and receive message payloads.

### Messaging

The messaging layer provides a simple API to perform WoT Thing operations such as InvokeAction, WriteProperty, etc. Most methods reflect the operations defined in the TD-1.1 specification. In addition, the SendRequest method is available.

This layer constructs a RequestMessage and passes it to the mapping layer. The mapping layer converts protocol messages back to responses and passes the response to the client.

For agents, the layer provides methods to construct a response and pass it to the mapping layer for further delivery using the underlying protocol.
With the response comes a status flag that can hold one of four values:

1. completed: The request was completed and no more responses are sent
2. failed: The request has failed and no more responses are sent. The error field holds an error title.
3. running: The request is running. Output optionally contains intermediate result or stream data.
4. pending: The request has been received but hasn't yet started. 

### Mapping

Mapping converts between RequestMessage/ResponseMessage, and the underlying protocol binding message.
This is implemented in accordance to the WoT specification of the protocol binding.

A special kind of mapping is the direct mapping. This is a simple pass-through of the request and response messages. As it bypasses the conversion overhead it should be the most efficient. 


### Client Transport

Client transports connect the client to the server and passing the provided messages using the underlying transport protocol.
Anything that can pass a message between client and server can act as a client transport.

Typically the transport and mapping are combined as described in the WoT Protocol Binding. However, it is also possible to use you own custom mapping and custom transport.

Transport clients implement the ITransportClient interface containing these methods:

1. SendRequest - send a request to the connected server.
2. SendResponse - send a response to the connected server.
3. SetConnectionHandler - succesful connections are reported to this handler:
4. SetRequestHandler - incoming request messages from the server are passed to this handler. (agents)
5. SetResponseHandler - incoming response messages are passed to this handler. (consumers)


### Server Transports

The server transport serves incoming connections initiated by the client. Once a connection is established the messages can be sent in either direction. Protocol messages are served by both client and server.

Server transports implement the ITransportServer interface containing these methods:
1. AddTDForms - add forms to the given TD for supported operations using this transport. 
2. GetForm - return the form for a specific operation. Primarily intended for testing.
3. GetConnectURL - provide the URL used to connect to this server
4. SendRequest - send a request to the connected client. Some operations require a subscription. (consumers)
4. SendResponse - send a response to the connected client (agents)
6. SetConnectionHandler - incoming connections are reported as an instance of the IServerConnection interface for sending messages to the remote client:
7. SetRequestHandler - incoming request messages are passed to this handler
8. SetResponseHandler - incoming response messages are passed to this handler


#### Authentication

TransportServers can provide facility to authenticate the client and MUST support the facility to verify identity before processing messages. Once a message is passed to the message handler, the client is considered to be authenticated. The only exception is the login request which is handled out of bounds, depending on the transport protocol. 

#### Discovery

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

## Direct Transport Server 

This transport establishes a connection using any means available. TCP, UDP, Bluetooth, IR, and of course HTTP/SSE_SC and Websockets. 

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
