# HiveOT Hub Core Design Overview

## Introduction


HiveOT stands for the "Hive of Things". It provides the means to collect and share sensor data and other information with its users. 

The Hive is a network of one or more Hubs that exchange this information. A Hub can be used to collect information for a geographical location, such as a home, or warehouse, or be dedicated to collection a type of information, for example environmental sensors from multiple locations.

Each Hub is linked to a collection of information sources such as IoT devices and services. Each source provides one or more 'Things' with IoT information. Sources connect to a Hub through a provisioning process (more on that below). The Hub handles the authentication and authorization of messages from and to the source. 

The hearth of a Hub is the 'Digital Twin Runtime' which includes support for various transport protocols. It manages a collection of Digital Twins for connected devices. A Digital Twin is a digital replica of an actual Thing. It contains the Thing Description (TD) document and property vales reflecting the state of a Thing. Modifying a property on a Digital Twin will be passed on to the actual device. Changes to the state of the device will be send as events to the Digital Twin which will forward it to subscribers. Device agents and consumers can use any of the supported protocol bindings to connect to the Hub and publish or retrieve Thing information.

![](digitwin-overview.jpg)

*Digital Twin Runtime Overview*

Thing information is sent to the Hub by so-called 'agents'. An agent can represent any number of Things. For security reasons there is no direct connection allowed between consumers and agents. Agents can therefore remain isolated from the consumers and connect using their own protocol. An agent will send the TD documents of each of its Things to the Hub, send events when Thing state changes and will receive actions directed at one of its Things by the digital twin service. The agent itself is also a Thing described with its own TD. Agents rely on the Hub for authentication and authorization. 

Agents need an account on the Hub before they are allowed to send or receive Thing information. The account can be provisioned manually, through a signed certificate, or automatically using the idprov service using an out-of-bounds verification mechanism.  

There are several types of agents:
* An IoT device that implements the HiveOT api or protocol directly.
* A protocol binding that bridges an existing IoT protocol to the HiveOT api or protocol. Example bindings are for IoT protocols zwave, zigbee, CoAP, and for external data sources such as weather stations, email, sms message and so on.
* Services that enrich information. For example the history service provides the ability to query historical event data obtained from Things. These services are also Things that have properties, send events and listen to actions. 

Users are consumers of information. Users subscribe to the Hub for events from Things and publish actions to trigger an action on a Thing. The authorization to subscribe to events and publish actions depends on the role the user has on the Hub. The Hub predefines roles for viewers, operators, managers and administrators. Additional custom roles can be defined with fine-grained access to specific sources.

An illustration of the event flow can be found here: [Event Flow](event-flow.jpg)

An illustration of the action flow can be found here: [Action Flow](action-flow.jpg)

In addition to receiving events and publishing of actions, users can receive a reply to an action on their local inbox address. When sending an action request, the users's reply-to address is used to receive a response. The reply-to address is that of a private inbox associated with the user's connection-id. The implementation depends on the protocol used to connect. When a user publishes an action request, the Hub digital twin forwards it to the agent for the Thing. The agent passes it to the device and sends the response back to the digital twin, which in turn passes it on to the user's inbox. Only messages that the agent and user are authorized for are passed on. The benefit of this approach is that the consumer and agent can use different protocols. The downside is the extra latency.

Services are agents that also support service RPC request-response style requests. The service configuration defines the roles that are allowed to invoke these service actions. For example, a user with the administrator role can publish the 'add user' action to the auth service. Special services can even require a custom role for them to be used.  

Finally, the Hub Core combines the digital twin, a store for TD documents, messaging protocols, authentication and authorization, into a light-weight stand-alone core service. The Hub can include any number of plugins for IoT protocol bindings and services such as a history service or a routing service to share information with other Hubs.


# Hub Protocols

The Hub includes several communication protocols out of the box: 
- HTTPS with SSE sessions using JWT/password authentication. 
- A simple REST API for login, token refresh and publishing events and actions.
- HTTPS with Websocket sessions (future)
- MQTT using mochi-co pub/sub messaging server using JWT/password authentication (future)

As all messages are routed via the hub, the hub can choose the optimal protocol to reach agent or consumer. The downside is extra latency as a message is transported twice, first to the Hub and then to the consumer or agent. The main benefit is that a consumer does not have to be aware of the protocol used by the device agent and vice versa.

Support for additional protocols can be added as needed. Note that the hub authentication mechanism must be supported by the protocol server. This currently includes username/password, jwt tokens and certificate based authentication.

Use of external protocol servers such as Redis can be accommodated with the caveat that account management must be synchronized between the authn service and the protocol server. Client account ID must match between the server and the runtime. Use of a 3rd party authentication server such as OAuth is currently not developed and need further investigation. 

### HTTPS/SSE/WS

The HTTPS/SSE/WS transport protocols open a listening TLS port for clients (both agents and consumers) to connect to. The connection handles publishing of events and actions, which are passed through the middleware and handled by the runtime as described below.

The transport configuration defines the addressing used for properties, events, actions, and the directory service. The address pattern used is that of /things/{thingID}/.... For example "/things/{thingID}/{key}. REST GET requests are used to read values while PUT requests are used to update values. Actions use a POST request to initiate an action including requesting a change in property configuration value.

These paths are included in the TD document forms, which describe how to access the values.
 
All requests must use TLS. Requests must carry a valid authentication token, except for the login request.

See the [HTTPS Binding Config](../runtime/protocols/httpsbinding/HttpsBindingConfig.go) for more details.

#### SSE

The server supports SSE sessions (Server Side Events) to notify consumers of events, and agents and services of actions.

The SSE connection path is and requires a valid session token:
* GET "/sse"

#### WebSockets (future)

The HTTP server supports websocket connections. The connection handles publishing of events, actions and rpc requests, which are passed through the middleware and handled by the runtime as described below.

WebSockets create a persistent connect (session) and use the message payload to identify the message. This uses the ThingValue object for exchange of messages. 

The WebSocket connection path is and requires a valid session token:
* GET "/wss"

#### RPC 

RPC support makes it a bit easier to invoke action requests on services and obtain
a reply. The HTTPS/SSE hubclient implements RPC support by including a messageID and wait for a delivery status messsage containing that message ID. This is implemented client side. The only requirement is that the agent that receives the agent sends a delivery status event containing the same message ID. The hub inbox intercepts the status events and forwards it to the consumer that sent the action request. 

### MQTT (in development)

The MQTT-5 message bus protocol supports both event style messages and RPC style using MQTT-5's reply-to facility. The protocol binding uses simple send-and-forget messaging and the RPC style for actions and rpc calls.

The message bus protocol binding connects Thing agents to the Hub and consumers to the Hub. Topic rules do not allow direct messaging between consumers and agents. The MQTT protocol binding can be used for one leg of the message to or from the Hub in case of multi-protocol routing, or both legs if the same protocol is used by agent and consumer. HiveOT uses the concept of an agent to help in authorization, selective subscription, routing and organizing Things by their agent. Agents are therefore part of the topic addressing scheme used. For communication, the hub is always one endpoint of the connection. At no time is a message from the agent directly send to the consumer or vice versa.  

Below a description of the topics used when passing messages between consumer, hub and agent.

Event Flow:
The event flow passes events published by Thing agents to listeners. The hub receives the events, updates the digital representation of the Thing and passes it on to the protocol handles for distributing it to authorized subscribers. 
1. Agent -> Hub
   PUB: "event/{agentID}/{thingID}/{key}/{agentID}" -> SUB: "event/+/+/#"
   * The Hub is authorized to listen to events from all agents

2. Hub -> Consumer  (sender is the hub, agent is in the thingID)
   PUB: "event/{hubID}/{agentID}/{thingID}/{key}" -> SUB: "event/{hubID}/{agentID}/{thingID}/{key}"
   * The Hub is authorized to publish events on its own ID
   * Consumers are authorized to subscribe to hub / agent events depending on their role. 

Action Flow:
1. Consumer -> Hub
   PUB: action/{hubID}/{agentID}/{thingID}/{key}/{consumerID}" -> SUB: "action/{hubID}/+/+/#"
   * The Hub listens for action requests made by consumers that are directed to the hub
   * The Hub can queue action requests, if configured, in case agents are intermittently connected
   * Consumers are not allowed to publish actions directly to the agent/device
   * Consumers are allowed to publish actions to the action / hubID depending on their role 
   * Future might see authorization for agent specific subscriptions 
   
2. Hub -> Agent (sender is hub)
   PUB: action/{agentID}/{thingID}/{key}/{hubID}"    -> SUB: "action/{agentID}/+/+/#"
   * The Hub sends queued action requests to the agent when it is connected.
   * Action requests are send in the order they arrive.
   * Agents are authorized to subscribe to actions for their own agentID
   * The Hub is allowed to publish actions for all agents

RPC Flow:
1. Consumer -> Hub
   PUB: "rpc/{hubID}/{agentID}/{interfaceID}/{method}/{consumerID}" ->   SUB: "rpc/{hubID}/+/+/#"
   * Mochi-co RPC wrapper uses the MQTT-5 reply-to address for an inbox. 
   * The hub client provides the reply it receives so that service and consumer can use different protocols.
   * For consideration is whether to allow direct calls if only a single protocol is used.
   * Requests are not queued. 
   * Consumers are authorized to publish RPC requests for services (agents) depending on their role. This is managed by the service configuration.
   * The Hub is authorized to subscribe to all rpc requests.
   * The Hub is authorized to publish a reply to the Consumer reply-to inbox address.
   
2. Hub -> Service
   PUB: "rpc/{agentID}/{interfaceID}/{method}/{consumerID}" ->   SUB: "rpc/{serviceID}/+/+/#"
   * The service listens for RPC requests and replies using the hub reply-to inbox.
   * The Hub is authorized to publish an rpc request (on behalf of a consumer) to all agents
   * Services are authorized to subscribe to rpc requests aimed at them
   * Services are authorized to publish a reply to the reply-to address.


## Hub Clients

Each transport protocol server includes a client that implements the IHubClient interface.

The Hub client implementations are for convenience. Each implementation supports methods for authentication, publishing events, TDs and action requests, subscribing to events, and set handlers for receiving events and action requests.

Use of this client is optional as consumers can use the TD forms to decide the protocol to use for invoke an action and subscribe to events. 

For convenience, client libraries are provided in Golang, Javascript and Python. (in development)

# Security

Security is a first-class feature of HiveOT. A significant level of protection is achieved by simply not allowing direct connections to IoT devices. Instead, agents and consumers authenticate and connect to the Hub and send their messages to the Digital Twin Runtime who routes the message. 

In addition, agents don't/shouldn't run servers. Instead, they connect to the Hub as a client. For 3rd party agents that do not support Hub protocols, a binding service can provide a bridge between device and the Hub. 

## Connectivity and Authentication

By default, all connections require TLS to ensure encrypted communication.

The Hub supports multiple forms of connection authentication depending on the communication protocol used. As a minimum a login/password, JWT token and pub/private key authentication are supported.

All users must be known to the Hub before they can connect with their credentials. Future consideration is the use of 3rd party auth providers.  

Authentication is managed through the embedded 'auth' service. 

## Authorization

Authorization determines which clients can subscribe to events and publish actions based on their role. This is controlled by the authorization service. 

### Agent Authorization

Agents connect to the Hub and have permissions to publish events with IoT data and receive action requests directed at things managed by the agent. 

Agents are authorized to:
  * publish a TD document for each of the Things it manages.
  * publish events for the Things it manages
  * subscribe to action request directed at things they manage 

As agents and Things are bound together, the hub 'digitwin' service combines these in a single ThingID in the format 'dtw:{agentID}:{thingID}'. When consumers read the Thing directory they will obtain the digital twin version of a Thing containing the digitwin ThingID. 

Actions to things are therefore send to the digital twin, not to the Thing agent directly. The digitwin 'inbox' receives a action requests for a Thing it determines the corresponding agentID. The inbox forwards the request to the connected agent and returns a delivery status message. When the agent receives the action it publishes a delivery event after the action is completed. The inbox intercepts this event and passes it on to the consumer that sent the action request.

If the agent is not directly available, the request can be queued for later delivery unless it expires in the inbox. The consumer should therefor not assume an immediate response as IoT devices can be offline or asleep for extended periods of time.


### Consumer Authorization

Consumer are clients that subscribe to events and publish actions. For simplicity they only have a single role on a hub. The role determines what message type a consumer can send or subscribe to. Predefined roles are viewer, operator, manager, and admin.

Users with the viewer and higher roles role can subscribe to events published by agents. Role authorization is given to subscribe to all events of all agents.

Users with operator roles can publish Thing actions.

Users with manager role can publish Thing actions and configure their properties.

Users with the admin role has the same permissions as the manager and can in addition use services that require the admin role. Each service sets the consumer roles can be be allowed or denied its use. See the documentation of the authz service for details.


#### Service Authorization

Services can play three different roles: that of an agent, a user, a service provider, or a combination of these.

Just like IoT devices, services are connected through agents. The agent publishes a TDD for each service that describes the supported methods actions. 

Example services are the digitwin directory, inbox, outbox, history store, state store, launcher, provisioning, rule automation. These are not IoT devices but can subscribe to events for further processing. Services can be interacted with through actions.

Service agents use the authorization service to configure a set of rules describing what roles can invoke actions on the service. For example, a client with the admin role can configure the history store while operators are not allowed to invoke these methods, event though both roles allow for publishing of actions. If no rules are defined than the standard roles apply.

The 'authz' authorization service is included in the hub runtime and consists of two services: administration and a user services. The administration service supports actions to manage roles while the user service lets a service agent set the permissions required to send actions to the service.

The authz agent therefore has two distinct capabilities, authorizing incoming messages and secondly, offering an API to manage the service. See the [authz TDD directory](../runtime/authz/tdd) for more details.


### Custom Roles (future consideration)

As described above, each role defines the allowed publish and subscribe addresses allowed by that role.

Custom roles allow fine-grained configuration of what sources and services the role can publish and subscribe to.

Custom roles can be used to provide more fine-grained permissions to consumers based on a specific use-case.

This is a future consideration.

