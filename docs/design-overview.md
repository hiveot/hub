# HiveOT Hub Core Design Overview

## Introduction


HiveOT stands for the "Hive of Things". It provides the means to collect and share sensor data and other information with its users. 

The Hive is a network of one or more Hubs that exchange this information. A Hub can be used to collect information for a geographical location, such as a home, or warehouse, or be dedicated to collection a type of information, for example environmental sensors from multiple locations.

Each Hub is linked to a collection of information sources such as IoT devices and services. Each source provides one or more 'Things' with IoT information. Sources connect to a Hub through a provisioning process (more on that below). The Hub handles the authentication and authorization of messages from and to the source. 

The hearth of a Hub is the 'Digital Twin Runtime' which includes support for various protocols. It manages a collection of Digital Twins for connected devices. A Digital Twin is a digital replica of an actual Thing. It contains the Thing Description (TD) document and property vales reflecting the state of a Thing. Modifying a property on a Digital Twin will be passed on to the actual device. Changes to the state of the device will be send as events to the Digital Twin which will forward it to subscribers. Device agents and consumers can use any of the supported protocol bindings to connect to the Hub and publish or retrieve Thing information.

![](digitwin-overview.jpg)

*Digital Twin Runtime Overview*

Thing information is sent to the Hub by so-called 'agents'. An agent can represent any number of devices. For security reasons there is no direct connection allowed between consumers and agents. Agents can therefore remain isolated from the consumers. An agent will send the TD documents of each of its Things to the Hub, send events when Thing state changes and will receive actions directed at one of its Things by the digital twin service. The agent itself is also a Thing described with its own TD. Agents rely on the Hub for authentication and authorization. 

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
- NATS/Jetstream pub/sub messaging server using NKeys and passwords for authentication
- MQTT using mochi-co pub/sub messaging server using JWT/password authentication
- HTTPS REST using JWT/password authentication. The rest API only supports polling of actions (for agents) and events (for consumers).
- Websocket sessions (future)
- HTTPS/SSE sessions (future)

As all messages are routed via the hub, the hub can choose the optimal protocol to reach agent or consumer. The downside is extra latency as a message is transported twice, first to the Hub and then to the consumer or agent. The main benefit is that a consumer does not have to be aware of the protocol used by the agent.  

Support for additional protocols can be added as needed. Note that the hub authentication mechanism must be supported by the protocol server. This currently includes username/password, jwt tokens and certificate based authentication.

Use of external protocol servers such as Redis can be accommodated with the caveat that account management must be synchronized between the authn service and the protocol server. Client account ID must match between the server and the runtime. Use of a 3rd party authentication server such as OAuth is currently not developed and need further investigation. 

## Agent vs Consumer protocols

Protocol servers separate the message flow from agents and consumers. This is managed through their addresses. 
It is required that all servers are able to authenticate the sender using the clientID registered with the authentication service. 

### HTTPS/SSE/WS

The HTTPS/SSE/WS protocol binding opens a listening TLS port for clients (both agents and consumers) to connect to. The connection handles publishing of events, actions and rpc requests, which are passed through the middleware and handled by the runtime as described below.

The binding configuration defines the paths used for properties, events, actions, and the directory service. The path pattern used is that of /subject/thingID/.... For example "/events/{thingID}/{key}. REST GET requests are used to read values while PUT requests are used to update values. Actions use a POST request to initiate an action including requesting a change in property configuration value.

These paths are included in the TD document forms, which describe how to access the values.
 
All requests must use TLS. Requests must carry a valid authentication token, except for the login request.

See the [HTTPS Binding Config](../runtime/protocols/httpsbinding/HttpsBindingConfig.go) for more details.

#### SSE

The server supports SSE sessions (Server Side Events) to notify consumers of events, agents of actions and services of rpc requests.

The SSE connection path is and requires a valid session token:
* GET "/sse"

#### WebSockets

The HTTP server supports websocket connections. The connection handles publishing of events, actions and rpc requests, which are passed through the middleware and handled by the runtime as described below.

WebSockets create a persistent connect (session) and use the message payload to identify the message. This uses the ThingValue object for exchange of messages. 

The WebSocket connection path is and requires a valid session token:
* GET "/wss"


### gRPC

The gRPC protocol binding opens a listening TLS port for gRPC clents (both agents and consumers) to connect to. The connection handles publishing of events, actions and rpc requests, which are passed through the middleware and handled by the runtime as described below.

gRPC creates a persistent connection (session) and depend on the message payload to identify the message. This uses the ThingValue object exchange messages. The connecting client generates a unique request ID for each message, based on the client's session ID and receives a response message with the same request ID. The approach taken is identical to that of the WebSocket binding.

The gRPC path is:
* "/grpc/{clientID}

Multiple gRPC connections with the same clientID are allowed.


### MQTT

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

The Hub implements two client implementations, the agent client for implementing agents and the consumer client for implementing consumers. These include the ability to discover the Hub, connect and authenticate. Agent clients can publish events and subscribe to actions while consumer clients can subscribe to events and publish actions. This client can make use of any of the available transports that implement the messaging protocol.

Use of the Hub clients is optional. The underlying messaging protocol can be used directly using the message definitions.

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

Authorization determines which clients can receive events and publish actions based on their role. This is controlled by the authorization service. Message bus communication protocols provide a first-line of defense authorization through configuration of allowed publish and subscribe addresses. The hub middleware can add additional authorization checks.

The included message bus protocol bindings follow the same addressing format that includes the message type, agentID, and thingID.  

Thing addresses are used for publishing events and receiving action requests of IoT devices and services in the format:

> {messageType}.{agentID}.{thingID}.{key}.{userID}

Where:
* '.' is the address path separator depending on the core used. Nats uses '.' while MQTT uses '/'.
* {messageType} is one of 'event', 'action', 'config', 
* {agentID} is the IoT device or service ID that manages Things.
* {thingID} is the ID of the Thing made available by the source.
* {key} is the name of the event, action or config field as defined in the Thing's TD.
* {clientID} is the account or login ID of the user that publishes action or config requests. A message bus protocol should ensure this matches the client's login ID.

A future consideration is the use of groups where events and actions can only be exchanged between group members. 


### Service RPC Addresses:

Some protocols support RPC style requests, which is useful for enrichment services such as directory, history and automation. The protocol driver sends the request and waits for a response before returning. This is intended for communication with services, not for IoT devices that might not always be connected. The result will be delivered to the senders inbox when received. The address format for these type of messages is similar to that of events and actions. The message type is that of 'rpc'. 

> rpc.{serviceID}.{capability}.{method}.{clientID}

Where:
* '.' is the address path separator depending on the core used. Nats uses '.' while MQTT uses '/'.
* rpc for invoking an RPC request on the service
* {serviceID} is the ID of the service that provides capability.
* {capability} is the name of the capability to use. Similar to interfaces.
* {method} is the name of the method to invoke as defined in the Service TD.
* {clientID} is the ID of the client that publishes RPC requests. Authorization is only granted to clients with the appropriate role. 

Note that RPC style messages are only allowed between consumer and hub, not between consumers or agents.

### Agent Authorization

Agents connect to the Hub to publish events with IoT data and subscribe to action requests.

Agents publish a TD document for each Thing they manage using an event with {key}='$td'. This TD document describes the properties, events and actions the Thing supports. The agent itself is also described in its TD document using the agentID as the ThingID. The HubClient provides an easy to use API for publishing a TD document.

Agents like IoT devices publish their events on the 'event' address and have permissions to publish events using the address:
> event.{agentID}.*.*   

Where the agentID must match the authentication ID of the agent device or service. '*' means that sources can publish any thingID and any event name. Mqtt uses '+'. 

Agents also have permission to subscribe to any action directed at the agemt, using the address:
> action.{agentID}.*.*.*

Where again agentID must match the authentication ID of the agent device/service. '*' means that sources subscribe to any thingID and action name, from any client. Mqtt uses '+'. 

When receiving an action request, agents reply with a response or error message send to the reply-to inbox address of the sender. This feature is natively supported in the NATS protocol. The MQTT implementation has added this feature using pub/sub of the INBOX address. See User authorization below for more information. Other protocols must provide their own method for handling replies. 

Agents can received queued actions to support intermittent connected devices. Queuing of actions is handled by the digital twin and is controlled by the action expiry time. 


### User Authorization

Users are consumers of information and thus subscribe to events and publish actions. For simplicity users only have a single role on a hub. The role determines what message type a user can send or subscribe to. Predefined roles are:

| role     | pub/sub message type    |
|----------|-------------------------|
| viewer   | SUB event               |
| operator | SUB event               |
| operator | PUB action              |
| manager  | SUB event               |
| manager  | PUB action, config      |
| admin    | SUB event               |
| admin    | PUB action, config, rpc |

Users with the viewer and higher roles role can subscribe to events published by agents. Role authorization is given to subscribe to all events of all agents using address wildcards:
> `event.*.*.*`

Users with operator or higher roles can publish Thing actions. These roles give authorization to publish on the address:
> action.{agentID}.{thingID}.{actionName}.{userID}

Authorization requires the {userID} part which must match the authenticated userID of the user publishing the action. It is used to validate the source of the request. 

#### Service RPC requests

Users can invoke RPC requests on services using address:
> rpc.{agentID}.{thingID}.{methodName}.{userID}
 
Where {thingID} is the name of the service capability or interface name, and {methodName} the name of the method to invoke. The authorization depends on the role as defined by the service configuration. 

To receive responses to rpc requests, users subscribe to their session inbox. The reply-to address permission is the subject. This follows NATS approach which has this behavior built-in:
> _INBOX.{sessionID}


### Service Authorization

Services can play three different roles: that of an agent, a user, a service provider, or a combination of these.

All services have the 'agentID' permissions for publishing events as an agent, identical to IoT device agents. They can publish events according to their TD and subscribe to actions.

Services also have user permissions based on their assigned role, identical to other users. This allows them to subscribe to events and publish actions.

Third, services can provide RPC capabilities that are restricted to certain roles. The service registers which roles are allowed to invoke a capability or method, which translates to a publish permission for users that have that role.
For example, an admin can publish rpc requests to a management capability that cannot be used by operators. 

Services use the 'rpc' prefix in their pub and sub addresses.
> rpc.{serviceID}.{capabilityID}.*.>

Services are configured in the authz (authorization) configuration. Each service capability is equivalent to a Thing and assigned to a role required to access them. For example:

| service   | capability | required role | purpose                                |
|-----------|------------|---------------|----------------------------------------|
| auth      | client     | admin         | manage devices, users and services     | 
| auth      | profile    | viewer        | assign/renew tokens                    | 
| auth      | roles      | admin         | manage and assign roles                |
| certs     | manage     | admin         | create client cert                     |
| provision | client     | norole        | request provisioning                   |
| directory | read       | viewer        | read things and their last known value |
| history   | read       | viewer        | read thing value history           |

The roles provide both publish action and subscribe event access on the service address.

Services actions are not queued and always confirmed with a response. If it fails the user should retry.

### Custom Roles (future consideration)

As described above, each role defines the allowed publish and subscribe addresses allowed by that role.

Custom roles allow fine-grained configuration of what sources and services the role can publish and subscribe to.

This capability can be used to give a user the ability to view or control specific Things without fully opening up all other Things as well.

This is a future consideration.

