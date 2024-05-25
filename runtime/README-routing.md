# HiveOT Runtime Routing

## Status 

In development

## Summary

The HiveOT runtime provides routing of events and actions through multiple protocols.
The runtime contains a digital twin service that consists of a Thing directory, value store and history store.

The protocol bindings authenticate request messages and pass them to the router. The router  invokes the middleware handlers, and if accepted passes the message to the registered handler for the message type.

Handling of messages depends on the type of message: 

### Events
Event messages follow a publish/subscribe approach. Thing agents publish an event while consumers subscribe to Thing events. Each can use their own protocol binding.

The general event flow is:
> thing -> agent -> protocol binding -> router -> [digital twin outbox]
>   [digital twin outbox] -> router -> protocol binding -> subscribers


Events delivered by Thing agents are stored in the digital twin's outbox. There is no reply other than the confirmation the event is received.

The outbox retains events until they expire which can vary between immediately to years.

Consumers can request an update from the outbox of a Thing for a given timeframe using the API supported by the protocol binding.

### Actions
Actions are messages targeted at a specific thing. Consumers publish action requests while agents subscribe to action requests. Each can use their own protocol binding.

The general action delivery flow is:
> consumer -> protocol binding -> router -> [digital twin inbox]
>   [digital twin inbox] -> router -> protocol binding -> agent -> thing
> 

The action status update flow:
> thing -> agent -> protocol binding -> router [digital twin inbox]
>   [digital twin inbox] -> router -> protocol binding -> consumer


Action requests are delivered by the protocol binding via the router to the digital twin inbox for the targeted Thing. Action requests always return with a delivery status containing the delivery progress and possibly a reply value. The action flow can hold one of the following delivery status values: 
* pending   - the action is received, placed in the outbox, but not yet delivered to the thing agent 
* delivered - the action request is delivered to the Thing agent but not yet applied
* waiting   - the action is waiting to be applied by the agent, eg device is asleep or offline
* applied   - the action was applied but result is waiting for confirmation
* completed - the action was applied and result is available
* failed    - unable to deliver the action to the agent or device

The action requests goes through two flows, the delivery flow and the return flow:
Steps of the delivery flow:
1. Consumer sends an action request via the protocol binding.
2. The protocol binding passes it to the router.
3. The router places the request in the digital twin inbox.
4a. If the agent is reachable, the request is forwarded to the protocol binding that delivers it to the agent. The request returns with the status 'delivered'.
4b. If the agent is not reachable, the request returns with the status 'pending'.
5. If an agent becomes connected then the router is notified who passes active requests currently waiting in the inbox to the agent and updates the delivery status to 'delivered'.

Steps of the return flow:
1. When the action for a Thing is applied and immediate feedback is received, the status message 'completed' is sent to the digital thing, containing the result value if applicable. 
2. If the action is applied and no immediate feedback is received then the delivery status message 'applied' is sent by the agent to the digital twin.
3. If the action is applied and fails then a delivery status message 'failed' is sent by the agent to the digital twin.
4. If the action cannot be applied because the Thing is offline or sleeping then the delivery status 'waiting' is sent with a reason code of 'sleeping' or 'offline'.
   4a. If waiting is not supported by the agent then the agent can send a failed status update and the consumer will have to send a new request. 
   4b. Once the Thing is reachable, the action is applied then this goes to step 2. 

Applying an action can result in a state change in the device. In this case the status change event is sent separately by the agent as described in the events section of the Thing's TDD.

When the digital twin receives a status update from the agent, it is passed to the consumer that sent the request, if it is online.

TBD: when an action request is received, the digital twin outbox could wait for a specified  time with replying to give the agent time to reply asynchronously with an action status update message. If received within a specified period then this status update is returned.


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

