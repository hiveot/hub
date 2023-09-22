# HiveOT Hub Core Design Overview

## Overview

HiveOT stands for the "Hive of Things". It provides the means to collect and share sensor data and other information between its users. 

The Hive is a network of Hubs that exchange this information. A hub can be used to collect information for a geographical location, such as a home, or warehouse, or be dedicated to collection a type of information, for example environmental sensors from multiple locations.

Each Hub is linked to a collection of information sources such as IoT devices and services. Each source contains one or more 'Things' that provide the IoT information. Sources connect to a Hub through a provisioning process (more on that below). The Hub handles the authentication and authorization of messages from and to the source. 

A Source publishes events with source information onto the message bus it connects to. Sources don't know who the consumers of their information are. There is no direct connection allowed from a consumer to a source. Each Source has the capability to connect to their linked Hub, send 'Thing Description' (TD) documents describing the capability of Things provided by the Source, send events containing Thing values and receive actions to interact with a Thing in the Source. The Source itself is also a Thing described with its own TD. 

Types of Sources:
* IoT device that implements the HiveOT protocol
* Protocol Binding that converts an IoT protocol, like zwave, zigbee, CoAP or internet obtained data such as weather, email, sms messages, to hiveot.
* Services that produce events.

Users are consumers of information. Users subscribe to events from Things and publish actions to interact with Things. The authorization to subscribe to events and publish actions to Things depends on the role the user has on the Hub. The Hub predefines roles for viewers, operators, managers and administrators. Additional custom roles can be defined with fine-grained access to specific sources.

In addition to receiving events and publishing of actions, users can receive a reply to an action on their inbox. When sending an action request, the users's reply-to address is used to receive a response. The reply-to address is that of a private inbox associated with the user's connection-id. (this is natively supported by the NATS message server). Sources can publish replies to any inbox while clients can only subscribe to their own inbox.

Services are sources that also support service RPC request-response style requests. The service configuration defines the roles that are allowed to invoke these service actions. For example, a user with the administrator role can publish the 'add user' action to the auth service. Special services can even require a custom role for them to be used. The message bus namespace for services are separate from those of IoT devices. See the following paragraphs for namespace definitions.  

Finally, the Hub Core combines the messaging server, authentication and authorization into a stand-alone core application. The Hub includes additional plugins for sources and services, such as a history service and a routing service to share information with other Hubs. 

## Hub Client

The Hub client aids the development of sources and user applications. It provides the ability to discover the Hub, connect to its message bus, subscribe to Thing events and publish actions.

Use of the Hub client is optional. The underlying messaging platform can be used directly using the message definitions.

For convenience, client libraries are provided in Golang, JS and Python. (in development) 

## Message Bus Protocols

The Hub's core supports multiple messaging bus protocols (ongoing development):
- NATS/Jetstream embedded server using NKeys and passwords for authentication
- NATS/Jetstream embedded server using callouts for JWT/password authentication
- MQTT using mochi-co embedded server using JWT/password authentication

Additional message bus servers can be added if needed. A Hub can only use a single messaging protocol at the same time, but can connect to another Hub that runs with a different protocol. 

Support for plain http, redis streams, rabbitmq, and others are options for the future.

# Security

Security is a first-class feature of HiveOT. This is provided by not allowing direct connections to IoT devices, support for hub firmware updates, and authentication and authorization integration with the message bus. 

## Connectivity and Authentication

By default, all connections require TLS to ensure encrypted communication.

The Hub supports multiple forms of connection authentication depending on the core it is running. As a minimum a login/password, JWT token and pub/private key authentication are supported.

All users must be known to the Hub before they can connect with their credentials. Future consideration is the use of 3rd party auth providers.  

Authentication is managed through the embedded 'auth' service. 

## Authorization

Authorization determines which clients can receive events and publish actions based on their role. This is controlled through message bus addresses.

Two types of addresses are in use, things and service addresses.

### Thing Source Addresses
Thing addresses are used for publishing events and receiving action requests of IoT devices and services in the format:

> {messageType}.{deviceID}.{thingID}.{name}.{userID}


Where:
* '.' is the address path separator depending on the core used. Nats uses '.' while MQTT uses '/'.
* {messageType} is one of 'event', 'action', 'config', 
* {deviceID} is the IoT device or service ID that manages Things.
* {thingID} is the ID of the Thing made available by the source.
* {name} is the name of the event, action or config field as defined in the Thing's TD.
* {userID} is the ID of the user that publishes action or config requests. Required when publishing action and config requests. Not applicable to events as the sourceID is the publisher of the event.

### Service RPC Addresses:

> rpc.{serviceID}.{capability}.{methodName}.{clientID}

Where:
* '.' is the address path separator depending on the core used. Nats uses '.' while MQTT uses '/'.
* rpc for invoking an RPC request on the service
* {serviceID} is the ID of the service that provides capability.
* {capability} is the name of the capability to use. Similar to interfaces.
* {methodName} is the name of the method to invoke as defined in the Service TD.
* {clientID} is the ID of the client that publishes RPC requests. Authorization is only granted to clients with the appropriate role. 


### Source Authorization

Sources publish events with IoT data and subscribe to action requests  .

Sources publish a TD document in an event with {name}='td' for each of the Things they provide. This TD document describes the properties, events and actions the Thing supports. The source itself is also described in its TD document using the sourceID as the ThingID.

Sources like IoT devices publish their events on the 'event' address and have permissions to publish events using the address:
> event.{deviceID}.*.*

Where the deviceID must match the authentication ID of the source device or service. '*' means that sources can publish any thingID and any event name. Mqtt uses '+'. 

Sources also have permission to subscribe to any action directed at the source, using the address:
> action.{deviceID}.*.*.*

Where again sourceID must match the authentication ID of the source device/service. '*' means that sources subscribe to any thingID and action name, from any client. Mqtt uses '+'. While it is possible for a source to limit receiving action to specific user clientID's, in most cases it will accept any clientID that can publish on the address.

When receiving an action request, sources reply with a response or error message send to the reply-to inbox address of the source. This feature is natively supported in the NATS protocol. The MQTT implementation has added this feature using pub/sub of the INBOX address. See User authorization below for more information.

Sources can received queued actions to support intermittent connected devices.  Queuing of actions is handled by the message bus and is controlled by the action expiry time. Normally the message bus will discard expired actions, but if a source does receive an expired action it must discard the request without replying.


### User Authorization

Users are consumers of information and thus subscribe to events and publish actions. For simplicity users only have a single role on a hub. The role determines what message type a user can pub/sub to. Predefined roles are:

| role     | pub/sub message type    |
|----------|-------------------------|
| viewer   | SUB event               |
| operator | SUB event               |
| operator | PUB action              |
| manager  | SUB event               |
| manager  | PUB action, config      |
| admin    | SUB event               |
| admin    | PUB action, config, rpc |

Users with the viewer and higher roles role can subscribe to events published by Sources. Role authorization is given to subscribe to all events of all sources using address wildcards:
> `event.*.*.*`

Users with operator or higher roles can publish Thing actions. These roles give authorization to publish on the address:
> action.{deviceID}.{thingID}.{actionName}.{userID}

Authorization requires the {userID} part which must match the authenticated userID of the user publishing the action. is used to track the source of the request. 

#### Service RPC requests

Users can invoke RPC requests on services using address:
> rpc.{serviceID}.{capabilityName}.{methodName}.{userID}
 
Where {capabilityName} is the name of the service interface and {methodName} the name of the method invoked. The authorization depends on the role as defined by the service configuration. 

To receive responses to rpc requests, users subscribe to their session inbox. The reply-to address permission is the subject. This follows NATS approach which has this behavior built-in!:
> _INBOX.{sessionID}

A core can have additional permission requirements:

### Service Authorization

Services can play three different roles: that of a Source, a User, and a Service provider, or a combination of these.

All services have the 'source' permissions, identical to IoT devices. They can publish events according to their TD and subscribe to actions.

Services also have user permissions based on their assigned role, identical to other users. This allows them to subscribe to events and publish actions.

Thirdly, services can provide RPC capabilities that are restricted to certain roles. The service registers which roles are allowed to invoke a capability or method, which translates to a publish permission for users that have that role.
For example, an admin can publish rpc requests to a management capability that cannot be used by operators. In this example the operator can publish to sources but not services.

To differentiate between access control of services from regular Sources, the Service use the 'rpc' prefix in their pub and sub addresses.
> rpc.{serviceID}.{capabilityID}.*.>

Services are configured in the authz configuration. Each service capability is assigned to a role required to access them. For example:

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


### NATS JetStream

The NATS core uses JetStream to stream and store events. This provides the capability of a directory and a history store. 

When using the NATS core, the directory and history services supports their features using NATS JetStream. On MQTT they manage their own store. 

When reading from the stream, users can open the 'events' stream by creating a JetStream 'stream consumer'.
To read a stream, consumers need permission to create a stream consumer son subject:
>  $JS.API.CONSUMER.CREATE|LIST|INFO.$events



