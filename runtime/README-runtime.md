# HiveOT Digital Twin Runtime 

## Status 

The runtime is in alpha. It is functional but breaking changes can be expected.

Detailed progress status below: 

### refactor hiveot messaging - complete

* hiveot messaging can be used separate from digital twin runtime [complete]
* protocols use the request/response messaging API [complete]
  * WoT http-basic [complete]
  * WoT websocket [spec in developement]
  * HiveOT websocket [complete]
  * HiveOT SSE [complete]
* client and server for supported protocols [complete]
* consumers and agents independent of the connection direction [complete]
* discovery of directory (with hiveot extensions) [complete]
 
### digital twin Thing directory [partial]
* CRUD directory [complete]
* compatible with WoT discovery and directory API [in progress]

### digital twin state management [partial]
* Store Thing and service configuration and state [complete]
* Services can persist their configuration in their digital twin [in progress]

### authentication [partial]
* token based authentication [complete]
* digest authentication [todo]
* oauth2 authentication [todo]

### authorization [partial]
* role based authorization [complete]
* service authorization [partial]
  * digitwin configuration integration [todo]

### routing between consumers and digital twin [complete]

* invoking actions [complete]
* read properties [complete]
* query actions [complete]
* subscribe events [partial]
    subscription [complete]
    retrieve recent events [todo]
* observing properties [complete]

### routing between digital twin and agents [partial]

* invoking actions [complete]
* read properties [complete]
* query actions [complete]
* subscribe events [partial]
    subscription [complete]
    retrieve recent events [todo]
* observing properties [complete]
* invoking actions
    * agent request flow [partial]
    * agent response flow [partial]


## phase 3 - refactor service plugins 

Service plugins:
- are extensions to the Hub and integral part of the runtime.
- run in their own process.
- can be started and stopped independently
- are managed through the launcher (also a plugin)
- store their configuration on their digital twin
  - publish changed configurations - digitwin stores it
  - read from digital twin ? how? different Thing IDs
- publish a TD
- include optional authorization 

* hiveoview [partial]
  * internal store for client state (remove state service)
  * publish a TD [partial]
  * manage configuration via digital twin [todo]

* certs
  * support for 3rd party certificate management
  * publish a TD [todo]
  * manage configuration via digital twin [todo]

* provisioning
  * redetermine what to do here

* launcher [complete]
    * publish a TD [partial]
    * manage configuration via digital twin [todo]
      * launching on startup
    * events for service start/stop/status [todo]

* history [partial]
  * store events [complete]
  * store property changes [complete]
  * store action history [todo]
    * store only stateful (unsafe) actions [todo] 
  * manage configuration via digital twin [todo]


## Summary

The HiveOT runtime provides routing of events and actions between devices, services and
consumers using one or multiple transport protocols. The runtime serves digital twins of 
IoT devices to consumers, containing the device Thing Definition and state. 

The runtime includes services for messaging, authentication and authorization and 
managing digital twin instances of IoT devices and services. It contains:
* Transport manager that aggregates protocol bindings for communication with devices, services and consumers
* Authentication service authenticates connections to the transport protocols
* Authorization service for authorizing the sending and receiving of messages by authenticated clients
* HubRouter action flow handles action requests from consumers and forwards it to the actual IoT devices and services
* HubRouter events flow receives events from IoT devices and forwards it to subscribers
* HubRouter properties flow receives property updates from IoT devices and forwards them to  subscribers
* Digitwin directory service serves the inventory of available devices with their digital twin Thing Definition

The hub differentiates three types of clients: IoT agents, consumers and services. Agents are clients that can operate on IoT devices or run stand-alone, and represent one or more IoT devices. They form a bridge between the native IoT device protocol and the Hub's WoT standards. Consumers are end-users that read information from the digital twins and send action requests to the digital twins. Services can act as agents and consumers.
Services enrich information received from the digital twin devices and publish the results.

The digital twin part of the runtime is designed using the W3C WoT standards and
handles Thing Description Documents, events, actions and properties.

### Events
Event messages follow a publish/subscribe approach. Thing agents publish events while consumers subscribe to Thing events. Each can use their own protocol binding.

The general event flow is:
> thing -> agent -> transport protocol -> [digital twin outbox]
>   [digital twin outbox] -> transport protocols -> subscribers

Events delivered by Thing agents are stored in the digital twin's outbox. There is no reply other than the confirmation the event is received by the Hub.

The outbox retains events until they expire which can vary between immediately to years. 

Consumers can request the latest event and property values of a Thing from the outbox.

![Event Flow](../docs/event-flow.jpg)

### Actions
Actions are messages targeted at a specific Thing or service. Consumers publish action requests while agents receive these requests. Each can use their own protocol binding.

The general action delivery flow is:
> consumer -> transport protocol -> [digital twin inbox]
>   [digital twin inbox] -> transport protocol -> agent -> Thing
> 

Agents send a status update when actions are applied and completed:
> Thing -> agent -> transport protocol -> [digital twin inbox]
>   [digital twin inbox] -> transport protocol -> consumer

![Action Flow](../docs/action-flow.jpg)

Action requests are delivered by the transport protocol binding to the digital twin inbox for the targeted Thing. Action requests always return with a delivery status containing the delivery progress and possibly a reply value. The action flow can hold one of the following delivery status values: 
* pending   - the action is received by the hub, placed in the outbox, but not yet delivered to the thing agent 
* delivered - the action request is delivered to the Thing agent but not yet applied
* waiting   - the action is waiting to be applied by the agent, eg device is asleep or offline
* applied   - the action was applied but result is waiting for confirmation
* completed - the action was applied and result is available
* failed    - unable to deliver the action to the agent or Thing

The action request goes through two flows, the delivery flow and the return flow:
Steps of the delivery flow:
1. Consumer sends an action request via its transport protocol.
2. The transport protocol passes it to the digitwin inbox.
3a. If the agent is reachable, the request is forwarded to the protocol binding that delivers it to the agent. The request returns with the status 'delivered'.
3b. If the agent is not reachable, the request returns with the status 'pending'.
4. When an agent connects then the inbox is notified who passes pending requests to the agent and updates the delivery status to 'delivered'.

Steps of the return flow:
1. When the action for a Thing is applied and immediate feedback is received, the status message 'completed' is sent to the digital twin inbox, containing the result value if applicable. 
2. If the action is applied by the agent and no immediate feedback is received then the delivery status message 'applied' is sent by the agent to the digital twin inbox.
3. If apply the action to the Thing fails, then a delivery status message completed is sent  by the agent to the digital twin inbox, containing the error message. This is a considered a successful delivery. Unfortunately it didn't work, hence the error.
4. If the action cannot be applied because the Thing is offline or sleeping then the delivery status 'waiting' is sent with a reason code of 'sleeping' or 'offline'.
   4a. If waiting is not supported by the agent then the agent can send a failed status update and the consumer will have to send a new request.
   4b. If the action expires before being applied then the agent drops the action and sends the failed status update with an error message describing it has expired.
   4c. Once the Thing is reachable, the action is applied then this goes to step 2. 

Applying an action can result in a state change in the device. In this case the status change event is sent separately by the agent as described in the events section of the Thing's TDD.

When the digital twin inbox receives a status update from the agent, it is passed to the consumer that sent the request, if it is online. If the consumer is offline then the status update is not repeated. On connecting the consumer can obtain the latest values from the outbox.

## Transport Protocols

A transport protocol is an embedded server that listens for incoming connections and authenticates the client. Transport protocols can also push messages to connected clients based on their subscriptions. 

Connectionless protocols are complemented with a connection-based return channel, or a polling features. 

If a connection based return channel is available, for example using SSE with HTTPS then the client's 'connected status' is based on the return channel. For example, the HTTPS protocol binding uses SSE as a return channel. SSE sessions are used to determine if the client is connected.

If a transport protocol does not support a return channel then it can fall back to polling on a prescribed interval. If polling doesn't take place within the interval, the connection is deemed lost. Some queuing is needed in this case to bridge the time between polling. This is specific to the protocol binding. 
  

## RPC requests

Some Hub clients use RPC style requests for reading available services. An rpc request is implemented as an action for the service and waiting for delivery update event containing the response.

RPC requests are implemented client side. When a consumer sends an RPC request, the protocol client waits for a delivery update event from the Hub before returning the result. This works independently from the protocol used by the agent.


## Built-in Services

The runtime includes essential built-in services to support the digital twin:
1. Authentication service. Intended for use by transport protocols to authenticate and verify a client's identity. The service client provides an RPC interface for managing clients including agents, consumers and services.
2. Authorization service. Intended for use by transport protocols to verify the ability of clients to publish or subscribe to certain agents, depending on the client's role.
3. Digital Twin services:
   a. The directory service stores TD documents
   b. The outbox service stores and forwards Thing events received from agents
   c. The inbox service stores and forwards Thing actions sent by consumers


---

## Q & A

1. When to use events vs properties? (option 2 - consumer centric)
    * Properties are used to report persisted internal device state.
        * Configurations are writable properties.
        * Immutable information are read-only properties.
        * Sensor status are read-only properties
        * Actuator status are read-only properties (control with actions)
    * Non-permanent or external state changes are reported with events
        * motion detection
        * doorbell button (and other push buttons)
        * actuator state changes are reported with events
        * sensor state changes are reported with events (the state is external)
    * Consumer notifications are events.
        * Alarms are always events.
        * Sensor updates are events as they are consumer focused. They can also be properties if they represent internal state. In this case they have the same name.
        * An actuator state change is an event as it is consumer focused. It also has a property that reflects the current actuator state.
        * events are recorded in the history store
    * Many events have corresponding properties. These should have the same name.
    * Events and writable properties have a history.
    * The final decision is up to the Thing agent.

2. When to use actions vs properties?
    * Consumer facing controls are always actions.
        * Differentiates from properties which are not consumer centric.
        * Actuators are always actions (lights, blinds, etc)
    * Stateless device operations are always actions
    * Action progress is reported with response messages and running status
    * If it is not idempotent (same changes, different outcome) it is an action.
    * Light/dimmer switch is a consumer action so an action.
    * Do not control a switch through a property.
    * If it requires multiple input parameters then it is an action.

3. How is action state stored in properties by digitwin?
   The state affected by actions are available through properties.
    * Option 1: as a single object property with nested properties for each value. Property name is the action name (just a convention)
    * Option 2: as individual properties. The action output is an object containing each affected property. (also just a convention)

4. How to track the action progress?
    * An action returns one or more response messages or action status messages. This is protocol dependent.
    * Response contains the progress status of the action using the http-basic specified states: pending, running, completed, failed
    * 'correlationID' is used to link the action with the response

5. How to track property write progress?
    * Property write operations return a response just like actions.
    * This is protocol dependent. Not all WoT protocols specify this.
    * In protocols that don't support acknowledgement, hiveot assumes completion on return.
    * Property values are not changed until observeproperty subscription sends an update.

6. How to integrate agents with the Hub
    * Agents connect to the hub, just like consumers
    * Agents serve requests and return responses. This is protocol dependent.
    * The Hub reconstructs a ResponseMessage contain the operation and correlationID and output. This is protocol dependent.

7. How do agents receive invokeaction and writeproperty requests?
    * The hub sends the request to the agent as if it is a consumer. The method is protocol dependent.
    * The agent returns an asynchronous ResponseMessage containing the correlationID and operation. This is protocol dependent.
    * The hub reconstructs the ResponseMessage when using non-hive message envelopes.
 


