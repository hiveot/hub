# HiveOT Messaging 

## Introduction

The HiveOT messaging package has the following objectives:
1. Support multiple protocols to exchange messages between consumers and Thing agents.
2. Standardize the messaging format independent of the underlying protocol.
3. Usable by the hiveot hub, its consumers and stand-alone Thing agents.

Note: the term 'messaging protocol' refers to any protocol that transport a request and response message between a client and server. In this context this means that http, sse, websocket, mqtt, etc are all different messaging protocols. 

In HiveOT the Hub acts as the Thing agent for digital twins. Consumers connect to the Hub and can use all the Things available on the Hub just like any other Thing agent. The Hub includes build-in services such as a digital twin directory and routing and authentication services. 

The hub itself is also a consumer of Things which are used to create a digital twin on the Hub. The main difference between hub and hiveot protocol agents is that hiveot agents connect to the Hub instead of the other way around. The messaging however is still that of a consumer (hub) talking to an agent. 

A future plan is to include a protocol binding to discover and add WoT devices on the network, as long as they are WoT compatible and offer a TD or TDD.

This messaging package is designed such that it can also be used stand-alone without requiring the Hub. A WoT thing can use the server to implement a WoT compatible device by listening to requests and sending responses. 

HiveOT standardizes the message envelope used by its messaging protocols. This envelope is mapped to the underlying WoT protocol for sending requests and receiving requests and responses. While HiveOT supports a number of WoT protocols, such as http-basic and websockets, it also includes endpoints that simply pass these envelopes as-is over http, websocket, sse and mqtt. 

This package has the following folder structure:
```
   messaging
     - clients      WoT protocol bindings for communicating with a WoT compatible server
     - servers      WoT protocol server for communicating with consumers
     - connections  Connection and subscription manager for use by servers
     - consumer     Consumer and agent helpers for using actions, properties and events.
     - tputils      Transport protocol helper classes and methods
     - tests        Test cases that operate on the messaging layer for testing protocol bindings. 
```


## Messaging on HiveOT

The messaging layer is a thin application layer built on the messaging protocols. It provides a simple WoT friendly API to consumers and Thing agents for building IoT applications and Things.

The purpose of this messaging layer is to enable WoT applications to work over any supported messaging protocol.

HiveOT messaging uses simple request-response-notification message envelopes. Clients send requests for operations, receive responses for these operations, and receive notifications to subscribed events and properties. It supports both direct and asynchronous transports. 

This approach builds on top of the WoT definition for 'operations' and adds operations for use by agents, supporting agents as clients of a hub or gateway. 

### Request Message

The purpose of the request message is for a client to send a request for an operation on a thing. Thing agents implemented using these messages are required to send a response when a request is received.

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

| name          | data type | description                                            | required  |
|---------------|-----------|--------------------------------------------------------|-----------|
| type          | string    | "request". Identifies the message as a request message | Required  |
| operation     | string    | Describes the request to perform                       | Required  |
| messageID     | string    | Unique identification of the message.                  | Required  |
| messageType   | string    | "request"                                              | Required  |
| correlationID | string    | Unique identifier of the request.                      | Optional  |
| thingID       | string    | ID of the thing the request applies to                 | Optional  |
| name          | string    | Name of the affordance the request applies to.         | Optional  |
| input         | any       | Input data as described by the affordance schema.      | Optional  |
| senderID      | string    | Authenticated sender of the request.                   | Optional  |


### Response Message

Responses serve to provide a client the progress or result of a request. A response message is linked to the request message through the correlationID field.

Response output is defined by the affordance in the TD.


| name          | data type | description                                              | required  |
|---------------|-----------|----------------------------------------------------------|-----------|
| type          | string    | "response". Identifies the message as a response message | required  |
| operation     | string    | The operation this is a response of.                     | optional  |
| messageID     | string    | Unique identification of the message.                    | required  |
| messageType   | string    | "response"                                               | Required  |
| correlationID | string    | identifies the request this is a response to.            | required  |
| thingID       | string    | ID of the thing the request applies.                     | optional  |
| name          | string    | Name of the affordance the request applies.              | optional  |
| output        | any       | Result of processing when status is "completed".         | optional  |
| status        | string    | Status of the request processing                         | required  |
| error         | string    | Error title if status is "failed".                       | optional  |
| timestamp     | string    | Timestamp the response was created                       | optional  |

### Notification Message

Notifications serve to provide a client with asynchronous updates of events, properties or actions. 

Notification data is defined by the corresponding affordance in the TD.

| name          | data type | description                                              | required  |
|---------------|-----------|----------------------------------------------------------|-----------|
| type          | string    | "response". Identifies the message as a response message | required  |
| operation     | string    | The operation this is a response of.                     | optional  |
| messageID     | string    | Unique identification of the message.                    | required  |
| messageType   | string    | "notification"                                           | Required  |
| correlationID | string    | identifies the request this is a response to.            | required  |
| thingID       | string    | ID of the thing the request applies.                     | optional  |
| name          | string    | Name of the affordance the request applies.              | optional  |
| output        | any       | Result of processing when status is "completed".         | optional  |
| status        | string    | Status of the request processing                         | required  |
| error         | string    | Error title if status is "failed".                       | optional  |
| requested     | string    | Timestamp the notification was created                   | optional  |

### Behavior

Any client can send a request. They can choose to wait for a response message or handle the response asynchronously. The HiveOT message envelope is identical regardless the underlying messaging protocol used such as http-basic, websocket, sse, mqtt and others. 

HiveOT uses simple message request, response, and notification message envelopes that the message protocols implement as their API. An additional 'agent' and 'consumer' helper builds these message envelopes for the various operations such as invoke action, subscribe events, write and observe properties.   

Requests are handled by the protocol server, which is run by the HiveOT Hub, or a Thing agent. When requests are received by the Hub, the hub digital twin responds to requests that read or query the digital twin state. Actions are forwarded to the actual Thing and the responses are returned to the client sending the request.

While this package is intended for connecting to the HiveOT Hub, it can also be used connecting between consumers and 3rd party Thing agents, including services. Use of the Hub is not a requirement for using this package.

When used with the Hub, these protocol clients support a reversal of connection, where they connect to the Hub instead of consumers connecting to the agent. Once the connection is established the message flow is the same between consumer and hub, and between hub and Thing agent. 

Consumers connect to the server using connection information described in TD forms. Agents connect to the Hub as a client using out-of-band token authentication. 

Hub or gateways will forward requests to the Thing agent. If the agent is not reachable at this point they return the error response or a pending response if there is support for queuing requests.

Thing agents will process the request. They SHOULD send a response with one of the statuses: pending, running, completed or failed. 

If a response with the 'running' status is sent, agents MUST also send a completed or failed response once the operation has finished.

If a hub or gateway is used then the response is received by the hub/gateway, which in turn updates the digital twin and passes the digital twin response to the client that sent the request. If the client is no longer reachable then the response can be queued or discarded, depending on the capabilities of the hub or gateway.

Clients will receive a response message containing the correlationID provided in the request, the payload, and any error information. Client implementations can choose to wait for a response (with timeout) or handle it as a separate callback.

When a client subscribes to an event or observes a property, it will receive ongoing events and property notifications. For action progress the status indicates the progress of the action, typically running.

## Design

There are 3 layers involved in exchanging messages:
1: message construction and decoding. HiveOT always uses RequestMessage and ResponseMessage envelopes.
2: mapping: map between request and response message envelope and the underlying protocol message types.
3: transport: establishing a connection to send and receive message payloads as per protocol specification.

Applications only need to implement the RequestMessage and ResponseMessage envelopes, or simply use the included Consumer and Agent helpers that provide a native API for these messages. This avoids a hard dependency between the application and a specific protocol, as WoT protocols use different message formats and behavior.

### Message construction and decoding - consumer sub-package

The messaging layer provides a simple application API to perform WoT Thing operations such as InvokeAction, WriteProperty, etc. Most methods reflect the operations defined in the TD-1.1 specification. In addition, the SendRequest method is available.

This layer constructs a RequestMessage, ResponseMessage and NotificationMessage envelope. NotificationMessage carry the status flag containing one of:
1. completed: The request was completed and no more responses are sent
2. failed: The request has failed and no more responses are sent. The error field holds an error title.
3. running: The request is running. Output optionally contains intermediate result or stream data.
4. pending: The request has been received but hasn't yet started. 

### Mapping - clients/servers sub-package

Mapping converts between RequestMessage/ResponseMessage envelopes and the underlying protocol implementation as per WoT specification.

A special kind of mapping is the direct mapping. This is a simple pass-through of the request and response messages. The idea behind this is to demonstrate the benefits of standardizing the message envelopes. 

For each underlying messaging protocol HiveOT supports passing the message envelopes directly. Obviously this isn't compatible with WoT protocol bindings but as it skips the mapping layer it is an efficient way of communicating between client and server. 


### Messaging Clients - clients package

Messaging clients implement the ITransportClient interface containing these methods:

1. SendRequest - send a request to the connected server.
2. SendResponse - send a response to the connected server.
3. SendNotification - send a notification to the connected server.
4. SetConnectionHandler - succesful connections are reported to this handler:
5. SetRequestHandler - incoming request messages from the server are passed to this handler. (agents)
6. SetResponseHandler - incoming response messages are passed to this handler. (consumers)
7. SetNotificationHandler - incoming notifications are passed to this handler

Client transports can be used by both consumers and agents. The use of client connection by agents is intended for agents that don't run a server and let the hub handle authentication and authorization.

### Messaging Servers - servers package

Messaging servers serves incoming connections initiated by the client. Once a connection is established the request and response messages can be sent in either direction. 

Servers implement the ITransportServer interface containing these methods:
1. AddTDForms - add forms to the given TD for supported operations using this transport. 
2. GetForm - return the form for a specific operation. Primarily intended for testing.
3. GetConnectURL - provide the URL used to connect to this server. Intended for use in discovery.
4. SendRequest - send a request to the connected client. Some operations require a subscription. (consumers)
5. SendResponse - send a response to the connected client (agents)
6. SendNotification - send a notification to the connected client.
7. SetConnectionHandler - incoming connections are reported as an instance of the IServerConnection interface for sending messages to the remote client:
8. SetRequestHandler - incoming request messages are passed to this handler
9. SetResponseHandler - incoming response messages are passed to this handler
10. SetNotificationHandler - incoming notifications are passed to this handler


#### Authentication

Messaging servers can provide the facility to authenticate the client and MUST support the facility to verify identity before processing messages. Once a message is accepted by a server and passed to the message request or response handler, the client is considered to be authenticated. One exception is the login request which is handled out of bounds, depending on the protocol. 

#### Discovery

Before a consumer or agent can connect to the server, the server must be discovered. 
The discovery server supports two modes of discovery:

1. WoT discovery protocol as described in [WoT Discovery](https://www.w3.org/TR/wot-discovery/)
This discovery method publishes the path to the TD of the directory service in a DNS-SD record. The directory service provides the TD's of available Things, which in turn contain forms that describe how to invoke operations on the Thing.

Additional methods can be used to share the Directory TD URL. HiveOT makes the directory TD available on the "/.well-known/wot" path as per specification.

TD's obtained through the directory will include forms that describe the protocols supporting operations.
For hiveot all TD operations are served by the digital twin so all forms will be identical.

2. HiveOT discovery
The hiveot discovery extends the DBS-SD WoT discovery with links to connection based protocols and use the instanceID of 'hiveot'. The hiveot connection protocols use the hiveot Request/Response message envelopes to exchange messages. These protocols do not use Forms in the TD and just use the 'base' field in the TD to identify the connection method and the 'links' field with alternative connection protocols.

The HiveOT discovery adds endpoint to the discovery TXT record with links to available endpoints:
Example TXT record: 
```
{
  "td": ".well-known/wot"
  "scheme": "https"
  "type": "Directory"
  
  "login": "/path/to/hiveot/login"
  "wss": "/path/to/hiveot/wss"
  "sse": "/path/to/hiveot/sse"
  "mqtts": "/path/to/hiveot/mqtt"
}
```
Except for login, all connection based protocols requires a valid authentication token.
The login endpoints supports digest authentication (in progress).

If the authentication token is missing the server will return a 401 with a WWW-Authenticate response 
header as described here: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/WWW-Authenticate


# Supported Messsaging Protocols

This section describes the supported messaging protocol details.

## HTTP-Basic  (WoT)

HTTP-Basic only supports synchronous request-responses from client to server and cannot pass asynchronous responses. It should therefore be used in combination with SSE.

Limitations:
1. There is no confirmation of an action on a remote Thing.
2. There is no output of an action on a remote Thing.
3. Subscription to events is not supported
4. Observing properties is not support.

Supported consumer operations:
* Invoke Action
  * Receive synchronous results only 
  * Results from remote Thing agents are not received 
* Write Property
  * Receive synchronous results only
  * Results from remote Thing agentsare not received
* Read Property and Read All Properties
  * Receive synchronous results only as provided by the hub digital twin
* Query Actions
  * Receive synchronous results only as provided by the hub digital twin
* Ping 

Support agent operations (extensions to WoT) :
* Send Response from agent to hub 
  * property update
  * events
  * actionstatus 

HttpServerTransport handles:
* receive requests - forward to handler
* receive notifications - forward to handler
* ping - synchronous response

Forms: 
The use of Forms is currently not compatible with the WoT specification as affordance level forms are not included. On the hub all forms are identical so it seems to be a lot of overhead for no gain. 

Instead only Thing level forms are included that use a generalized href for all operations:
> /http/{operation}[/{thingID}[/{name}]]

Where {operation}, {thingID} and {name} are URI variables and only used when applicable.

The method used depends on the operation as described in the HTTP-Basic specification.

Example:

To read the property value of a thing:
> GET https://server/http/readproperty/{thingID}/{name}

These operations are described in a TD top level form as per [TD-1.1 specification](https://www.w3.org/TR/wot-thing-description11/#form):
```json
{
  "forms": [
    {
      "op": ["readproperty"],
      "href": "/http/{operation}/{thingID}/{name}",
	  "htv:methodName": "GET"
    }
  ]
}
```


### HTTP+SSE-SC (HiveOT)

This messaging protocol combines http-basic with a single SSE return channel for all messages, subscriptions and observations from the connected client. It uses the hiveot RequestMessage and ResponseMessage envelopes which contain the operation, thingID, affordance name, input or output, correlationID and timestamp.  

HTTP methods are used to (un)subscribe events and (un)observe properties. These are defined in the Forms.

Limitations:
1. This is not a WoT SSE subprotocol specification (maybe it should be)
2. The application must use the hiveot message envelopes (or this library). 
3. HTTP headers are used to define a connection ID ["cid"] to correlate http requests with the SSE connection that sends the replies.

Example:

Forms to connect, subscribe and unsubscribe events of a Thing could look like:
```json
{
  "forms": [
    {
      "op": "subscribeallevents",
      "href": "/ssesc/subscribeallevents/{thingID}",
      "htv:methodName": "GET",
      "subprotocol": "sse-hiveot",
      "headers": ["cid"]
    },
    {
      "op": "observerproperty",
      "href": "/ssesc/observerproperty/{thingID}",
      "htv:methodName": "GET",
      "subprotocol": "sse-hiveot",
      "headers": ["cid"]
    }
  ]
}
```

To subscribe to events, post to the subscribe endpoint with the thingID. This returns with a 200 code on success or 401 when the consumer is not allowed subscriptions to this Thing.

Include a 'cid' header field containing a connection-id. This is required for linking the subscription to the SSE connection that receives the events or property updates. .

Once a request is received, an agent has to send a response. When the agent is another client, as in hiveot, the response is sent via the Hub and must include the correlationID provided with the request. 

### HTTP+WebSocket (WoT) 

The WoT draft websocket protocol uses HTTP to establish an authenticated websocket connection. All further messages are exchanged over this websocket connection.
The WoT draft proposal defines separate message envelopes for each action, property, and event request and responses.

Limitations:
1. The specification is draft and subject to change.
2. It might not fully support the needed responses.
3. Forms use is provisionary and likely incomplete.

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

### HTTP+WebSocket (HiveOT)

The HiveOT websocket protocol uses HTTP to establish an authenticated websocket connection. All further messages are exchanged over this websocket connection.
HiveOT simply passes the RequestMessage and ResponseMessage envelopes over the websocket connection and doesn't need to do any mapping.

Limitations:
1. This is not a WoT standard


## MQTT Transport Server (todo)

The HiveOT Hub runs a MQTT broker. Consumers and agents connect to the broker MQTT over TCP or Websocket. 

Discovery publishes the MQTT address, TPC and Websocket ports.

This is not intended for connecting to 3rd party brokers. This would require a separate protocol binding

## MQTT Transport Bridge (todo)

Intended to interoperate with 3rd party WoT devices using a 3rd party MQTT.


## CoAP Transport (todo)

this is under investigation

## Discovery 

HiveOT discovery provides a means for consumers and agents to locate Things and Thing Directory. the Hub and query the directory of Things. It follows the W3C draft specification: https://w3c.github.io/wot-discovery/ 

The Hub uses DNS-LD to publish a WoT directory on the local network and supports the ".well-known/wot" endpoint to read the directory TD.

HiveOT also adds the available Hub endpoints to facilitate direct connection with the supported hiveot protocols without the use of forms. The endpoints are added to the DNS-SD TXT record with their schema:
* wss: hiveot websocket connection endpoint that uses the hiveot message envelopes 
* sse: hiveot sse-sc connection endpoint that uses the hiveot message envelopes
* mqtt: (in progress) mqtt broken that that uses the hiveot message envelopes

## Authentication

The Hub maintains a list of clients, their credentials and roles. In order to authenticate, the client ID and credentials must be known.

All transport bindings support the use of authentication tokens to authenticate a connection. The HTTP binding provides a login method to obtain and refresh a token. The hubcli also offers an out-of-band way to generate authentication tokens.

Authentication using client certificates are planned for the future.

OAuth2 support is planned for the future. 
Radius support is planned for the future

## Connecting to 3rd Party WoT devices

Since the Hub is also a Thing agent, Thing consumers that implement http-basic or wot websockets can work with the hub. They are served digital twin Things through the exposde directory.

Thing agents are implemented through protocol bindings that connect to the Hub. The Hub is based on the concept that IoT devices (agents) connect to it with the intent to reduce the attack surface of IoT devices by eliminating the servers altogether. As an added benefit there is no need to implement authentication on these devices. The downside is that these devices are not WoT compatible.

For consumers there is no difference between connecting to a Thing directly or to the HiveOT Hub. The only requirement is that they know that the Hub is a gateway for multiple Things, so they need to read the directory after connecting. From there on, Forms in the TD will provide the Hub endpoint to connect to. Any interaction is WoT compatible, depending on the protocol used. 

For stand-alone Things the situation is different as they expect consumers to connect to them. To interoperate with these devices, the Hub implements a WoT binding (todo) that discovers WoT compatible devices and connect to them using configured credentials. The Hub acts as the consumer for these devices. (This binding is not yet implemented.)
