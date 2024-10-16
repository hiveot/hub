# Notes on transport protocols

## The HTTP transport protocol

The HTTP transport protocol is divided in a basic and sub-protocols.

### HTTP Basic

It is rather simple for a server to listen for post and get commands over https and return the requested information. The path contains the thing ID and optionally property, event and action name. For example:

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

HTTP basic alone however is insufficient. To receive subscribe to events, observe properties and for agents to receive action requests and property write requests, a sub-protocol is needed that allows the server to push messages to the consumer or agent.    

### WoT SSE Sub-protocol (not yet supported)

The WoT SSE subprotocol provides a method to return subscribed events and observed properties over the SSE connection. 

A form that could subscribe to all events of a Thing looks like:
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

Note that every single subscription requires a new connection. This makes it virtually useless for use in browsers when connecting using http/1.1 as browsers have a 6 connection limit to a single endpoint. Http/2 theoretically supports connection sharing and allow for many SSE connections to the same endpoint. Whether this is much better is debatable as every consumer would require one or more SSE connections for every single Thing that is available on the hub or gateway. So, 1000 Things, 10 consumers, 1 subscribeallevents per thing and 1 observedallproperties adds up 20000 SSE connections as a best-case scenario. This, obviously, doesn't scale well.

Due to these mentioned limitations, HiveOT does not currently support the WoT SSE sub-protocol. As a minimum it should be possible to use the connection to receive messages from multiple things and support multiple subscriptions over this connection. 

### Http SSE-SC Sub-protocol

The HiveOT SSE-SC (SSE shared-connection) sub-protocol defines a way for a single SSE endpoint to be shared by all interaction affordances of one or more web things. 

This is not a WoT sub-protocol and intended to overcome the limitations of the WoT SSE sub-protocol by allowing interactions for all Things to take place over a single SSE connection regardless whether they use HTTP/1.1 or HTTP/2.

It behaves more akin to the websocket sub-protocol in that event subscription and observe properties requests are posted by the consumer to the server. The subscribed events and properties are returned over the single existing connection. Just like with WS this adds the need to identify the resource whose information is sent.  

A form to subscribe to all events of a Thing could look like:
```json
{
  "forms": [
    {
      "op": "sse:connect",
      "href": "/sse",
      "htv:methodName": "GET",
      "subprotocol": "ssesc",
      "headers": ["cid"]
    },
    {
      "op": "subscribeallevents",
      "href": "/ssesc/digitwin/subscribe/dtw:agent1:thing1",
      "htv:methodName": "POST",
      "subprotocol": "ssesc",
      "headers": ["cid", "messageID"]
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


### Http WS Sub-protocol

The WoT Websocket sub-protocol works is a HTTP protocol extension to allow both requests and subscriptions of all interaction affordances to take place over a single websocket connection.

A form to subscribe to all events of a Thing could look like:
```json
{
  "forms": [
    {
      "op": "subscribeallevents",
      "href": "/ws/digitwin/subscribe/dtw:agent1:thing1",
      "subprotocol": "ws"
    },
    {
      "op": "unsubscribeallevents",
      "href": "/ws/digitwin/unsubscribe/dtw:agent1:thing1",
      "subprotocol": "ws"
    }
  ]
}
```

This follows the specification from the 'strawman proposal' https://docs.google.com/document/d/1KWv-aQfMgsqBFg0v4rVqzcVvzzisC7y4X4CMUYGc8rE by Ben Francis. See also webthings.io: https://webthings.io/api/#protocol-handshake. 

```json
{
  "messageType": "{operation-name}",
  "topic": "event/{thingID}/{name}",
  "topic": "property/{thingID/{name}",
  "topic": "action/{thingID/{name}",
  "topic": "td/{thingID}",
  "data": "..."
}
```
Where topic is one of the four options describing the payload:
* event is a message from Thing {thingID} with affordance name {name}.
* property is a message from Thing {thingID} with affordance name {name}.
* action is a message for Thing {thingID} with affordance name {name}.
* td is a TD document from Thing {thingID}

All messages are send via the websocket connection. Events are therefore sent to the websocket connection that passed the subscription request. 

This is under development.


# HTTP Implementation (under development)

Http server listens on 8443
register routes for http requests.
- each request carries an auth token
- auth token contains client ID and session ID

ssesc subprotocol:
* http binding -> [N]sub-protocol bindings
* -> ws binding
* -> sse binding
* -> sse-sc binding
*    manage connections:
*     SSE-Connection:
*        -> [header] -> connectionID
*        -> [auth token] -> clientID, sessionID
*        -> add/remove subscription -> []subscriptions
*        -> invoke action to agent
*        -> publish event, publish property to consumer
*     Connect/disconnect: add/remove connection in map;
*        How to determine connectionID from clientID/sessionID?
*          cid = sessionID - one connection per session (no browsers) 
*                optional cid or tabid header for browsers
*        Multiple connections from one or more consumed things? 
*          Yes: consumedthings adds cid=sessionID.thingID header
*               subscription requests 
*        auth token contains client type as agent?
*        -> map[agentID]SSE-Connection
*        -> map[consumerID.connectionID]SSE-Connection
*     InvokeAction: 
*       1. determine agentID and thingID
*       2. lookup connection of agent -> how?
*           A: iterate all connections - how many to expect? 
*           B: separate map of [agentID]connectionID (agents have 1 connection)
*       3. InvokeAction on connection
* 
*     manage subscriptions: -> A manage inside connection
*     -> subscribe/unsubscribe (sessionID,connectionID,thing,name) 
*     PublishEvent/Prop (dThingID,name):
*       A: Manage subscription inside the connection
*          If connection has subscription for (dThingID,name)
*            then PublishEvent/Prop on connection
*          pro: easy to manage. removing connection also removes subscription
*          con: publish needs to iterate all connections
*       B: Map of [subscription dTthingID.name] -> []connectionID
*          pro: Efficient. Immediately find all connections to publish to
*          con: adding/removing a connection needs iteration of all subscriptions
    

- [connection-id] -> []connection
- or
- binding: map[connectionID] -> connection
-  multiple sessions per client
-  multiple connections per session 
- client -> session -> connection
  - how do subscription requests identify the connection?
  - A: by sessionID 
    - issue: multiple browser tabs share the same subscription
  - B: by connectionID  
    - issue: subscriptions need to identify the connectionID
    - connection header field for connectionID
    - default to sessionID
    - ending a session?