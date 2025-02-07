# HiveOT Hub Core Design Overview (in development)

## Introduction


HiveOT stands for the "Hive of Things". It provides the means to collect and share sensor data and other IoT information with its users. 

The Hive is a network of one or more Hubs that exchange this information. A Hub can be used to collect information for a geographical location, such as a home, or warehouse, or be dedicated to collection a type of information, for example environmental sensors from multiple locations.

Each Hub is a consumer of one or more information agents such as IoT devices and services. Each agent provides one or more 'Things' with IoT information. Agents connect to a Hub through a provisioning process (more on that below). The Hub handles the authentication and authorization of messages from and to the agents. 

The hearth of a Hub is the 'Digital Twin Runtime' which contains Digital Twins of Things and supports interaction through various transport protocols. A Digital Twin is a digital replica of an actual Thing. It contains the Thing Description (TD) document and property, event and action state values of the Thing. Writing a property on a Digital Twin will be passed on to the actual device (exposed Thing). Changes to the state of the device will update the digital twin, which in turn notify consumers. Consumers can use any of the supported protocol bindings to connect to the Hub and retrieve Thing information.

![](digitwin-overview.jpg)

*Digital Twin Runtime Overview*

Thing information is obtained by the Hub from so-called 'Thing Agents'. An agent can manage one or any number of Things. There is no direct connection allowed between consumers and agents. Consumers work with digital twins while the Hub works with the actual Thing provided by agents. Agents therefore remain isolated from the consumers and connect using their own protocol. After an agents connects, the Hub  discovers the TD's of each of its Things, observes Thing properties, subscribes to Thing events and send requests to write properties or invoke actions. The agent itself can also provide a Thing for itself, for example to manage its configuration.  
Agents need an account on the Hub before they are allowed to send or receive Thing information. The account can be provisioned manually, through a signed certificate, or automatically using the idprov service using an out-of-bounds verification mechanism.  

Consumers typically login using username and password that are setup with the consumer's account. On successful login a consumer is issued an authentication token that is used for the actual authentication. Authentication tokens can also be issued through other identify verification means such as a client certification and (in future) oauth.   

There are several types of agents:
* An IoT device that implements the HiveOT api or protocol directly is called an exposed Thing and acts both an agent and a Thing.
* An IoT protocol binding that bridges an existing IoT protocol to the HiveOT hub. Example bindings are for zwave, zigbee, CoAP protocols, and for external data sources such as weather stations, email, sms message and so on.
* A service that enriches information. For example, the history service provides the ability to query historical event data obtained from Things. 

Consumers are users of information. Consumers subscribe to the Hub for events from Things and publish actions to trigger an action on a Thing. The authorization to subscribe to events and publish actions depends on the role the consumer has on the Hub. The Hub predefines roles for viewers, operators, managers and administrators. Additional custom roles can be defined with fine-grained access to specific sources.

An illustration of the event flow can be found here: [Event Flow](event-flow.jpg)

An illustration of the action flow can be found here: [Action Flow](action-flow.jpg)

In addition to receiving events and publishing of actions, consumers can receive a progress updates of an action through action status messages. When a user publishes an action request, the Hub digital twin forwards it to the agent for the Thing. The agent passes it to the device and sends the response back to the digital twin, which in turn passes it on to the consumer. Changes to the state of the device are send by the device as property updates to the digital twin, which in turn passes it on to observers of that property. This approach allow consumers to be notified of time-delayed updates that result from actions. 

In addition, the digitwin runtime offers an API to read the most recent actions and their delivery status. This lets the consumer view when the action was delivered, which is especially important for devices that are only intermittently reachable.

Services combine the capabilities of agents and consumers. They can publish events and actions, receive actions, and subscribe to events through the service agent. Services authenticate using a predefined authentication token, which is generated by the launcher.
On startup, services register the roles that are allowed to invoke actions on the service. For example, the included history service requires the viewer role for sending the action to read the history and manager role to configure the history.  

The Hub digital twin runtime combines the digital twin service, store, messaging protocols, authentication and authorization into a light-weight stand-alone runtime. The Hub can include any number of plugins for IoT protocol bindings and supporting services such as a history service. Plugins register their TD after connecting to the hub via one of the available protocols.  

Last but not least, the plan is for a bridge service that supports exchange of specific IOT data with other hubs, creating a hive of things.


# Hub Transport Protocols

The Hub includes several transport protocols out of the box. A transport protocol can support consumers and agents with endpoints for reading, writing and observing of properties, invoking actions and subscribing to events from digital twin Things. 

When digitwin TD documents are requested, these protocols inject 'Forms' that describe the support operations.

The Hub also includes a client library for various programming languages that provides a consumed-thing implementation for consumers and a exposed-thing implementation for use by agents.

Initially the HTTPS/SSE transport using JWT/password authentication is supported out of the box. The return channel uses server-sent-events which is a lightweight uni-directional communication channel that runs over HTTP.

Additional transport protocols are planned:
- HTTPS using Websockets sessions  
- MQTT using pub/sub messaging server using JWT/password authentication 
- Other pub/sub message bus protocols being considered are NATS, Redis, and HiveMQT.

Support for additional transport protocols can be added as needed. Note that the hub authentication mechanism must be supported by the protocol server. This currently includes username/password, jwt tokens and certificate based authentication.

Use of external protocol servers such as Redis can be accommodated in the future with the caveat that account management must be synchronized between the authn service and the protocol server. Client account ID must match between the server and the runtime. Use of a 3rd party authentication server such as OAuth might be supported in the future but is currently out of scope. 

## HTTPS/SSE/WS protocols

The HTTPS transport protocol open a listening TLS port for clients (both agents and consumers) to connect to. The server supports both the SSE and Websocket sub-protocols for establishing two-way communication. 

The transport injects Forms into the digitwin TD document describing the endpoints for operations for using properties, events and actions on the Thing's digital twin.  

All requests must use TLS. Requests must carry a valid authentication token, except for the login request.

#### SSE Sub-protocol

The server supports the SSE (Server Side Events) sub-protocol for notification of consumers of events, and agents and services of actions.

The SSE connection path is included in the TD Thing level forms. Connecting to the SSE enpoint requires a valid session token.

#### WebSocket Sub-protocol (in progress)

A websocket sub-protocol is in the works.

#### RPC 

RPC support makes it a bit easier to invoke action requests on services and obtain a reply synchronously. 

The HTTPS/SSE client implements RPC support by including a correlationID and wait for a delivery status event containing that message ID. The agent that receives the action sends a delivery status event containing the same message ID. The digitwin service receives the delivery status event events and forwards it to the consumer that sent the action request. 

This RPC capability is not part of the WoT standard but does use WoT specified mechanism of actions and events.

### MQTT (planned)

MQTT and other message bus protocols support publish and subscribe sessions natively. The message bus server must integrate with the hub's authentication and authorization services to ensure property security.

The Mqtt protocol binding injects Forms in the digitwin TD that describes how to perform operations on the digitwin Thing, as described in the WoT specification.

### CoAP (planned)

CoAP protocol support is planned for the future. This can be implemented as an embedded transport protocol or as a separate Agent plugin for CoAP devices which can be located separate from the Hub. The agent uses the http or mqtt transport protocol to connect to the Hub. This is to be decided. 

## Hub Clients

For each transport protocol server clients are provided that implement the IHubClient interface. These client support Form operations for their respecting protocol implementation and can be used with the consumed-thing and exposed-thing instances for agents, services and consumers, or used directly. 

The Hub client implementations are for convenience. 3rd party WoT compliant clients can be used with the Hub instead. 

# Security

Security in HiveOT is a first-class multi-layered feature of HiveOT.  
The first layer of protection is provided by simply not allowing direct connections between consumers and IoT device agents. Instead, agents and consumers authenticate and connect to the Hub and send their messages to the Digital Twin Runtime who routes the message. 

The second layer of protection is provided by 'requiring' that IoT device agents do not listen on any network ports, unless this is required for their functionality. Instead, agents connect to the Hub as a client.
For 3rd party IoT devices that do not support HiveOT rules, a protocol binding plugin can provide a bridge between device and the Hub, allowing the IoT device to be placed on a separate secured network segment or behind a firewall.

The third layer of protection is the using of short-lived JWT authentication tokens for consumers, that are linked to their session. If a valid token is obtained by another client then it cannot be used as each connection has its own session. Tokens expire within weeks. Consumers must reconnect before expiry to refresh their token, or login with their password again. This is a consumer facing security. Services and agents use a single persistent session but are still required to periodically refresh their authentication token.

The fourth layer of protection is provided through role based authorization. Consumers are only able to access Things based on their role. A viewer role does not allow sending actions to Things for example.

Middleware adds additional protection against abuse by include rate control and optionally additional security checks. (this is still under development)


## Connectivity and Authentication

By default, all connections require TLS to ensure encrypted communication.

Out of the box, the hub uses a self-signed certificate. For local this can be sufficient as long as consumers have access to the CA certificate. Support for lets-encrypt is planned for the future.

The Hub supports multiple forms of connection authentication depending on the communication protocol used. As a minimum a login/password, JWT token and pub/private key authentication are supported.

All users must be known to the Hub before they can connect with their credentials. Future consideration is the use of 3rd party auth providers.  

Authentication is managed through the embedded 'authn' service. 

## Authorization

Authorization determines which clients can subscribe to events and publish actions based on their role. This is controlled by the authorization service. 

### Agent Authorization

Each agent is only allowed to publish on their own behalf.

Agents connect to the Hub and have permissions to publish events with IoT data and receive action requests directed at things managed by the agent. 

Agents are authorized to:
  * publish a TD document for each of the Things it manages.
  * publish events for the Things it manages.
  * update property values for the Things it manages.
  * receive action request directed at things they manage. 

When a Thing agent publishes a TD, events, and property changes, the digital twin services modifies the thingID with a digital-twin thingID (dThingID) prefix containing 'dtw:{agentID}'. The digital thing ID thus becomes 'dtw:{agentID}:{thingID}'. When consumers read the Thing directory they will obtain the digital twin version of a Thing containing this digitwin ThingID.

On startup, agent implementations can register their own authorization rules with the roles that are allowed to use the things or services they offer. The hub middleware applies these rules to consumers that use the digital twin Thing.

### Consumer Authorization

Consumers are clients that subscribe to events, observe properties, and publish actions. Consumers have a role that determines whether they are allowed to publish messages containing actions and configuration properties. For simplicity consumers only have a single role on a hub. Predefined consumer roles are viewer, operator, manager, and admin.

Consumers with the viewer and higher roles role can subscribe to events published by IoT agents.  

Consumers with operator roles can publish Thing actions.

Consumers with manager role can publish Thing actions and Thing properties to configure Things.

Consumers with the admin role has the same permissions as the manager and can in addition use services that require the admin role. Each service sets the roles that are allowed or denied its use. See the documentation of the authz service for details.


#### Service Authorization

Services can play three different types of roles: that of an agent, a consumer, a service provider, or a combination of these.

Just like IoT devices, services are connected through agents. The agent publishes a TDD for each service that describes the supported  actions. 

Bundled services are the digitwin directory, digitwin authentication service, history store, state store, launcher, and provisioning services. Service methods are invoked by sending an action to the corresponding service dThingID.

The 'authz' authorization services is built into the hub runtime. It provides capabilities for administration authorization and validating authorization, which are defined in their TD document.


### Custom Roles (future consideration)

As described above, each role defines the allowed publish and subscribe addresses allowed by that role.

Custom roles allow fine-grained configuration of what sources and services the role can publish and subscribe to.

Custom roles can be used to provide more fine-grained permissions to consumers based on a specific use-case.

This is a future consideration.

