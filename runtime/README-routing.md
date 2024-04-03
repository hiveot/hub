# HiveOT Runtime Routing

## Status 

In development

## Summary

The objective of HiveOT runtime is to provide routing of events, actions, and rpc requests through multiple protocols.
The runtime comes with a digital twin service that consists of a Thing directory and value store.

The protocol bindings authenticates request messages and passes them to the router. The router first invokes the registered middleware, and if accepted passes the message to the registered handler for the message type.


The general incoming message flow is:
> client -> protocol -> router -> middleware 
>                              -> handler [queue]  => protocol -> client

The result of the handler is returned to the protocol binding. The protocol binding sends it as a reply to the client. 

Handling of messages depends on the type of message: 

* Event messages follow a publish/subscribe approach. Device agents publish an event while consumers subscribe to events. Events can be published with or without delivery guarantee. The protocol binding confirms the delivery as soon as the event is received and handed off to the router. The router forwards events to active subscribers. Events are not queued. All protocol bindings must support publishing of, and subscribing to event messages.

* Action messages delivered by the protocol binding are passed to the router which returns a delivery status response containing a request ID. Delivery status updates are sent until delivery is complete or cancelled. 
  * The handler first attempts to immediately deliver the message by passing it to the protocol bindings. If a protocol binding has an active session then the message is passed to the destination agent and a response is received. The handler returns a delivery status message containing the agent's response. 
  * If none of the protocol bindings can deliver the message then delivery is queued. When the destination connects, the message is delivered and the response is sent to the original sender's inbox using a delivery status message. If the message expiry time has passed it is removed from the queue. All protocol bindings must support action messages. Status updates are sent as events to the caller's inbox containing the request ID. 

* RPC style messages are handled similar to action message except that they are not queued. 


## Protocol Binding

A protocol binding is an embedded server that listens for incoming connections and authenticates the client. Protocol bindings can also push messages to connected clients based on their subscriptions. 

Connectionless protocols are complemented with a connection-based return channel, or a polling features. 

If a connection based return channel is available, for example using SSE with HTTPS then the connection status is based on the  return channel. For example, the HTTPS protocol binding uses SSE as a return channel.

If a protocol binding does not support a return channel then it can fall back to polling on a prescribed interval. If polling doesn't take place within the interval, the connection is deemed lost. Some queuing is needed in this case to bridge the time between polling. This is specific to the protocol binding. 
  

## Router

The router handler receives messages from the protocol bindings. The router first passes the message through the middleware stack. The middleware stack handles logging, authorization, rate control and other steps. Each step in the middleware can abort the message handling.

Next the router passes the message to the handler associated with the message type.

The event message handler forwards the event to all active protocol bindings. The protocol binding will push it to connected clients that have subscribed to the sending agent, thingID or event key as described in the thing TDD.

The action message handler identifies the agent the message is directed at. If the agent has an active session then the request is forwarded to the agent using its active session. It then responds to the request with a 'delivered' status message containing the response payload. If the agent has no active session then the message can be queued with an expiry time as defined by the agent. 

The rpc message handler identifies the agent the message is directed at. If the agent has an active session then the request is forwarded to the agent. The agent forwards it and receives a response. The response is sent back to the original caller. If the agent does not have an active session a not-connected response is returned to the original sender.

## Built-in Services

The runtime includes the following built-in services that the router is aware of:
1. Authentication service. Intended for use by protocol bindings to verify a client's identity. An RPC interface provides the ability to manage clients including agents, consumers and services.
2. Authorization service. Intended for use by protocol bindings to verify the ability of clients to publish or subscribe to certain agents, depending on the client's role.
3. Digital Twin service. Is used by the router to capture TD documents and Thing value updates. An RPC interface provides the ability to query the directory and Thing's current values.



## External Services

Services register their RPC endpoints by publishing a TD describing the available interface and method names. The service instance is the agent. Each interface corresponds to a Thing ID. The method names are defines as actions. The Thing @type attribute defines the type of service using the hiveot vocabulary.

