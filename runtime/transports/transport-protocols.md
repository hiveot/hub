# Notes on transport protocols

## The https transport protocol

The http transport protocol is divided in a basic and one or more sub-protocols.

### Https Basic

It is rather simple for a server to listen for post and get commands over https and return the requested information. The path contains the thing ID and optionally property, event and action name. For example:

To return the TD of a thing:
> GET https://server/digitwin/td/{thingID}

To read a Thing property value:
> GET https://server/digitwin/property/{thingID}/{name}
 
These operations can easily be described in a TD top level form as per [TD-1.1 specification](https://www.w3.org/TR/wot-thing-description11/#form):
```json
{
  "forms": [
    {
      "op": "readproperty",
      "href": "/digitwin/property/dtw:agent1:thing1/{name}",
	  "htv:methodName": "GET"
    }
  ]
}
```

The real challenge comes to support consumers subscribing to events and observing properties. In addition, in case of the HiveOT hub, Thing agents also need to receive actions and write-property requests, as they don't run servers.

### WoT SSE Sub-protocol (not supported)

The WoT SSE subprotocol provides a method to return subscribed events and observed properties over the SSE connection.

Note that every single subscription requires a new connection. This makes it virtually useless from browsers when connecting using http/1.1 as browsers have a 6 connection limit to a single endpoint.

http/2 theoretically supports connection sharing and allow for many SSE connections to the same endpoint. Whether this is a wise thing to do is up for debate as it does take a large amount of resources. Especially in case of a hub or gateway, every consumer would require one or more SSE connections for every single Thing that is available on the hub or gateway. So, 1000 Things, 10 consumers, 1 subscribeallevents per thing and 1 observedallproperties adds up 20000 SSE connections. This, obviously, doesn't scale well.  

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

Due to the mentioned limitations, and without an applicable use-case, HiveOT does not support the WoT SSE sub-protocol.

### Http SSE-SC Sub-protocol

The HiveOT SSE-SC (SSE shared-connection) sub-protocol defines a way for a single SSE endpoint to be shared by all interaction affordances of one or more web things. 

This is not a WoT sub-protocol and intended to overcome the limitations of the WoT SSE sub-protocol by allowing all interactions for all Things to take place over a single SSE connection regardless whether they use HTTP/1.1 or HTTP/2.

It behaves more akin to the websocket sub-protocol in that event subscription and observe properties requests are posted by the consumer to the server. The subscribed events and properties are returned over the single existing connection. Just like with WS this adds the need to identify the resource whose information is sent.  

A form to subscribe to all events of a Thing could look like:
```json
{
  "forms": [
    {
      "op": "sse:connect",
      "href": "/sse",
      "htv:methodName": "GET",
      "subprotocol": "sse-sc"
    },
    {
      "op": "subscribeallevents",
      "href": "/sse-sc/digitwin/subscribe/dtw:agent1:thing1",
      "htv:methodName": "POST",
      "subprotocol": "sse-sc"
    },
    {
      "op": "unsubscribeallevents",
      "href": "/sse-sc/digitwin/unsubscribe/dtw:agent1:thing1",
      "htv:methodName": "POST",
      "subprotocol": "sse-sc"
    }
  ]
}
```
This introduces a 'connect' operation for establishing an sse connection. This operation is only needed once for all subscriptions. This establishes an SSE connection as per RFC or returns 401 when the consumer does not provide proper credentials connecting to the sse endpoint.

To subscribe to events, post to the subscribe endpoint with the thingID. This returns with a 200 code on success or 401 when the consumer is not allowed subscriptions to this Thing.


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

The only other specification is webthings.io: https://webthings.io/api/#protocol-handshake. This is an old specification and it is unclear if it is still valid. It contains an envelope with 'messageType' and 'data' fields. As these don't include a thingID and affordance name, HiveOT adds a 'topic' field that matches the mqtt binding.

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


When subscribing, the consumed-thing client connects to the href URL if a connection doesn't yet exist.
* the consumed thing returns a subscription object to the consumer which can be used to unsubscribe.
* when observing or subscribing this provides the data as described by the property or event's dataschema. 
* The payload contains:

? yeah, how does this work?
? use 'additional '