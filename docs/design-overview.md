# HiveOT Hub Design Overview (in development)

## Introduction


HiveOT stands for the "Hive of Things". It provides the means to collect and share sensor data and other IoT information with its users. 

The Hive is a network of one or more Hubs that exchange this information. A Hub can be used to collect information for a geographical location, such as a home, or warehouse, or be dedicated to collection a type of information, for example environmental sensors from multiple locations.

Each Hub is a consumer of one or more information agents such as IoT devices and services. Each agent provides one or more 'Things' with IoT information. Agents connect to a Hub through a provisioning process (more on that below). The Hub handles the authentication and authorization of messages from and to the agents. 

The heart of a Hub is the 'Digital Twin Runtime' which contains Digital Twins of Things and supports interaction through various transport protocols. A Digital Twin is a digital replica of an actual Thing. It contains the Thing Description (TD) document and property, event and action state values of the Thing. Writing a property on a Digital Twin will be passed on to the actual device (exposed Thing). Changes to the state of the device will update the digital twin, which in turn notify consumers. Consumers can use any of the supported protocol bindings to connect to the Hub and retrieve Thing information.

![](digitwin-overview.jpg)

*Digital Twin Runtime Overview*

Thing information is obtained by the Hub from so-called 'Thing Agents'. An agent can manage one or any number of Things. There is no direct connection allowed between consumers and agents. Consumers work with digital twins while the Hub works with the actual Thing provided by agents. Agents therefore remain isolated from the consumers and connect using their own protocol. After an agents connects, the Hub  discovers the TD's of each of its Things, observes Thing properties, subscribes to Thing events and send requests to write properties or invoke actions. The agent itself can also provide a Thing for itself, for example to manage its configuration.  
Agents need an account on the Hub before they are allowed to send or receive Thing information. The account can be provisioned manually, through a signed certificate, a token, or automatically using the idprov service using an out-of-bounds verification mechanism.  

Consumers typically login using username and password that are setup with the consumer's account. On successful login a consumer is issued an authentication token that is used for the actual authentication. Authentication tokens can also be issued through other identify verification means such as a client certification and (in future) oauth.   

There are several types of agents:
* An IoT device that implements the HiveOT api or protocol directly is called an exposed Thing and acts both an agent and a Thing.
* An IoT protocol binding that bridges an existing IoT protocol to the HiveOT hub. Example bindings are for zwave, zigbee, CoAP protocols, and for external data sources such as weather stations, email, sms message and so on.
* A service that enriches information. For example, the history service provides the ability to query historical event data obtained from Things. 

The hub acts as an agent to consumers, who subscribe to the Hub for events from digital twin Things and publish actions to trigger an action on a Thing. The authorization to subscribe to events and publish actions depends on the role the consumer has on the Hub. The Hub predefines roles for viewers, operators, managers and administrators. Additional custom roles can be defined with fine-grained access to specific sources.

An illustration of the event flow can be found here: [Event Flow](event-flow.jpg)

An illustration of the action flow can be found here: [Action Flow](action-flow.jpg)

When publishing an action, consumers receive response messages until the action has completed. When a user publishes an action request, the Hub digital twin passes it to the agent for the Thing. The agent passes it to the device and determines a running or completed response. The response is send back to the digital twin, who uses the correlationID to match it with the original request. The digital twin then passes it on to the consumer. The Hub API uses the standardized RequestMessage and ResponseMessage envelopes while the underlying transport protocol such as http-basic or websocket binding maps requests and response to their protocol specific counterpart.  

Changes to the state of the device are send by the device as property updates to the digital twin, who observes all property changes. The digital twin updates its state and then notifies the observers of the digital twin with the new value.  

The digitwin runtime supports operations to read the most recent property value, event value and action status. The underlying transport binding translates the request to the protocol specific format and does the reverse for the response..

Services combine the capabilities of agents and consumers. They can publish events and actions, receive actions, and subscribe to events through the service agent. Services authenticate using a predefined authentication token, which is generated by the launcher.
On startup, services register the roles that are allowed to invoke actions on the service. For example, the included history service requires the viewer role for sending the action to read the history and manager role to configure the history.  

The Hub digital twin runtime combines the digital twin service, store, messaging protocols, authentication and authorization into a light-weight stand-alone runtime. The Hub can include any number of plugins for IoT protocol bindings and supporting services such as a history service. Plugins register their TD after connecting to the hub via one of the available protocols.  

Last but not least, the plan is for a bridge service that supports exchange of specific IOT data with other hubs, creating a hive of things.


# Hub Transport Protocols

The Hub includes several transport protocols out of the box. A transport protocol can support consumers and agents with endpoints for reading, writing and observing of properties, invoking actions and subscribing to events from digital twin Things. 

When digitwin TD documents are requested, these protocols inject 'Forms' that describe the supported operations.

The Hub also includes a client library for various programming languages that provides a consumed-thing implementation for consumers and a exposed-thing implementation for use by agents.

Initially 3 transport protocols are supported: The hiveot HTTPS/SSE-SC, hiveot websocket, and WoT websocket (strawman) transports. The hiveot protocols use the standard RequestMessage and ResponseMessage envelopes as-is while the WoT websocket (strawman) transport maps to the woT websocket message types. The MQTT protocol server and client bindings will be added next. 

Use of external protocol servers such as Redis can be accommodated in the future with the caveat that account management must be synchronized between the authn service and the protocol server. Client account ID must match between the server and the runtime. Use of a 3rd party authentication server such as OAuth might be supported in the future but is currently out of scope.

Last but not least, agents connect to the hub instead of the other way around. While this connection reversal is independent from the role of consumer and agent, it does mean that the client side must be able to handle requests from the hub to subscribe to events and observe properties.    

## HTTPS/SSE/WS protocols

The HTTPS transport protocol open a listening TLS port for clients (both agents and consumers) to connect to. The server supports both the SSE and Websocket sub-protocols for establishing two-way communication. 

The transport injects Forms into the digitwin TD document describing the endpoints for operations for using properties, events and actions on the Thing's digital twin.  

All requests must use TLS. Requests must carry a valid authentication token, except for the login request.

#### HiveOT SSE Sub-protocol

The hub server supports a modified SSE (Server Side Events) single-connection sub-protocol. This sub-protocol, named 'sse-hiveot', uses the RequestMessage and ResponseMessage envelopes. These envelopes include the operation, thingID and affordance name. This enables the use of a single sse-connection to pass requests and responses for different operations, Things and affordances. To subscribe to events and observe properties the http endpoint is invoked as defined in the Form generated by this subprotocol.   


#### HiveOT WebSocket Sub-protocol

The HiveOT websocket sub-protocol simply passes the RequestMessage and ResponseMessages as-is over the connection. Requests to subscribe to events and observe properties are handled by the protocol binding.

#### WoT WebSocket Sub-protocol

The WoT websocket protocol specification is currently being reviewed and is subject to change. This transport protocol implementation currently implements a message mapping from RequestMessage to WoT websocket message types. The reverse takes place for ResponseMessages. 

#### Consumer 

The golang library includes a Consumer type for use by, .. well, consumers. The Consumer provides an API for supported operations and converts the request to a standed RequestMessage. The reverse happens on the Response side. A consumer instance works with any hiveot transport protocol binding.  

Consumer applications can use the Consumer instance for convenience. It also supports an RPC api that matches asynchronous responses with their request using the message correlationID. 

### MQTT (planned)

MQTT and other message bus protocols support publish and subscribe sessions natively. The message bus server must integrate with the hub's authentication and authorization services to ensure property security.

The Mqtt protocol binding injects Forms in the digitwin TD that describes how to perform operations on the digitwin Thing, as described in the WoT specification.

### CoAP (planned)

CoAP protocol support is planned for the future. This can be implemented as an embedded transport protocol or as a separate Agent plugin for CoAP devices which can be located separate from the Hub. The agent uses the http or mqtt transport protocol to connect to the Hub. This is to be decided. 

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

Out of the box, the hub uses a self-signed certificate. For local usage in edge devices this can be sufficient as long as consumers have access to the CA certificate. Support for lets-encrypt is planned for the future.

The Hub supports multiple forms of connection authentication depending on the communication protocol used. As a minimum a login/password, JWT token and pub/private key authentication are supported.

All clients must be known to the Hub before they can connect with their credentials. Future consideration is the use of 3rd party auth providers.  

Authentication is managed through the embedded 'authn' service. 

## Authorization

Authorization determines which clients can subscribe to events and publish actions based on their role. This is controlled by the authorization service. 

### Agent Authorization

Agents connect to the Hub. After authentication, the hub subscribes to the agents to receive events and observe properties. Yes, this is done over a reverse connection. 

Agents must use a pre-provided auth token or certificate. They can refresh the token and store it for later use. 

On startup, agents can register their own authorization rules with the roles that are allowed to use the things or services they offer. This service is provided by the authz service. The agent invokes the action of the authz service Thing. The hub middleware applies these rules to consumers that use the digital twin Thing.

### Consumer Authorization

Consumers are clients that subscribe to events, observe properties, and publish actions. Consumers have a role that determines whether they are allowed to publish messages containing actions and configuration properties. For simplicity consumers only have a single role on a hub. Predefined consumer roles are viewer, operator, manager, and admin.

Consumers with the viewer and higher roles role can subscribe to events published by IoT agents.  

Consumers with operator roles can publish Thing actions.

Consumers with manager role can publish Thing actions and Thing properties to configure Things.

Consumers with the admin role has the same permissions as the manager and can in addition use services that require the admin role. Each service sets the roles that are allowed or denied its use. See the documentation of the authz service for details.


#### Service Authorization

Services can play three different types of roles: that of an agent, a consumer, a service provider, or a combination of these.

Just like IoT devices, services are connected through agents. The agent publishes a TD for each service that describes the supported actions. 


### Custom Roles (future consideration)

As described above, each role defines the allowed publish and subscribe addresses allowed by that role.

Custom roles allow fine-grained configuration of what sources and services the role can publish and subscribe to.

Custom roles can be used to provide more fine-grained permissions to consumers based on a specific use-case.

This is a future consideration.

