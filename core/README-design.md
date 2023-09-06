# HiveOT Hub Core Design Overview

## Overview

HiveOT stands for the "Hive of Things". It provides the means to collect and share sensor data and other information with users such as people or software services. 

The Hive is a network of Hubs that exchange this information. A hub can be used to collect information for a geographical location, such as a home, or warehouse, or be dedicated to collection a type of information, for example environmental sensors from multiple locations.

Each Hub is linked to a collection of information sources. Each source contains one or more 'Things' that are the source of IoT information. Sources are linked to a Hub through a provisioning process (more on that below). The Hub handles the authentication and authorization of messages from and to the source. 

A Source publish events with source information. Sources don't know who the consumers of their information is. There is no direct connection allowed to a source. Each Source has the capability to connect to their linked Hub, send 'Thing Description' (TD) documents describing the capability of Things provided by the Source, send events containing Thing values and receive actions to interact with a Thing in the Source. The Source itself is also a Thing described with its own TD. 

Types of Sources:
* IoT device that implements the HiveOT protocol
* Protocol Binding that converts an IoT protocol, like zwave, zigbee, CoAP or internet obtained data such as weather, email, sms messages, to hiveot.
* Services that process data from other sources and produce events with the result.

Users are consumers of information. Users subscribe to events from Things and publish actions to interact with Things. The authorization to subscribe to events and publish actions to Things depends on the role the user has on the Hub. The Hub predefines roles for viewers, operators, managers and administrators. Additional custom roles can be defined with fine-grained access to specific Sources or Things.

In addition to receiving events and publishing of actions, users can receive a reply to an action on their inbox. When sending an action request, the users's reply-to address is used to receive a response. The reply-to address is that of a private inbox associated with the user's connection. (this is natively supported by the NATS message server).

Services are Sources that in addition support service actions in a request-response style. The service configuration defines the roles that are allowed to invoke these service actions. For example, a user with the administrator role can publish the 'add user' action to the auth service. Special services can even require a custom role for them to be used. 

Finally, the Hub Core combines the messaging server, authentication and authorization into a stand-alone core application. The Hub includes additional sources and capabilities such as a history service and a routing service to share information with other Hubs. 

## Hub Client

The Hub client aids the development of sources and user applications. It provides the ability to discover the Hub, connect to its message bus, subscribe to Thing events and publish actions.

Use of the Hub client is optional. The underlying messaging platform can be used directly using the message definitions.

For convenience, client libraries are provided in Golang, JS and Python. (in development) 

## Message Bus Protocols

The Hub's core supports multiple messaging bus protocols (ongoing development):
- NATS/Jetstream embedded server using NKeys and passwords for authentication
- NATS/Jetstream embedded server using callouts for JWT/password authentication
- MQTT using mochi-co embedded server using JWT/password authentication

Support for other messaging servers can be added. A Hub can only use a single messaging protocol at the same time, but can connect to another Hub that runs with a different protocol. 

Support for plain http, redis streams, rabbitmq, and others are considered for the future.

# Security

## Connectivity and Authentication

All connections require TLS to ensure encrypted communication.

The Hub supports multiple forms of connection authentication depending on the core it is running. As a minimum a login/password, JWT token and pub/private key authentication is supported.

Authentication can be managed through the 'auth' Service.

## Authorization

Authorization determines which clients can receive events and publish actions based on their role.
This is managed by limiting access to message bus addresses.

The address format looks like:
> {prefix}.{sourceID}.{thingID}.{messageType}.{name}.{clientID}

Where:
* '.' is the address path separator depending on the core used. Nats uses '.' while MQTT uses '/'.
* {prefix} is either 'things' for sources or 'svc' for services.
* {sourceID} is the source or service authentication ID.
* {thingID} is the ID of the Thing made available by the source, or the provided capability of the service.
* {messageType} is one of 'event', 'action', 'config'
* {name} is the name of the event, action or config field as defined in the Thing's TD.
* {clientID} is client sending the action or config request. Required for publishing all actions.


### Source Authorization

Information Sources publish events and subscribe to action requests.

Sources publish a TD document in an event with {name}='td' for each of the Things they provide. This TD document describes the properties, events and actions the Thing supports. The source itself is also described in its TD document using the sourceID as the ThingID.

Sources publish their events on the 'things' event address. User roles control permission for users to subscribe to events and publish thing actions. 

Sources have permissions to publish events using the subject:
> things.{sourceID}.{thingID}.event.{name}

Sources have permission to subscribe to any action directed at the source, using the subject:
> things.{sourceID}.*.action.>

Actions published can be queued to serve intermittent connected devices. Actions can be given an expiry time after which they are discarded when not delivered. An expiry of 0 means that the action will not be queued if the source is offline. Availability of this feature can depend on the core used.


### User Authorization

The user role determines what message type a user can pub/sub to. Predefined roles are:

| role     | prefix      | message type     | pub or sub |
|----------|-------------|------------------|------------|
| viewer   | things      | event            | sub        |
| operator | things      | event            | sub        |
| operator | things      | action           | pub        |
| manager  | things      | event            | sub        |
| manager  | things      | action, config   | pub        |
| admin    | things      | event            | sub        |
| admin    | things, svc | action, config   | pub        |


Users with the viewer and higher roles role can subscribe to events published by Sources. Role authorization is given to subscribe to all events of all sources using address wildcards:
> `things.*.*.event.*`

Users with operator or higher roles can publish Thing actions. These roles give authorization to publish on the address:
> things.{sourceID}.{thingID}.action.{actionName}.{userID}

Where {userID} is required and must match the authenticated userID of the user publishing the action. This is used to track the source of the request. 

Users can invoke actions on services using address:
> svc.{serviceID}.{capabilityName}.action.{actionName}.{userID}
 
Where {capabilityName} is the name of the service interface and {actionName} the name of the method invoked. The authorization depends on the role as defined by the service configuration. 

To receive responses to actions, users subscribe to their session inbox. The reply-to address permission is the subject. This follows NATS approach which has this behavior built-in!:
> _INBOX.{userID}.>


A core can have additional permission requirements:

### Service Authorization

Services can play three different roles: that of a Source, a User, and a Service provider, or a combination of these.

All services have the 'source' permissions, identical to IoT devices. They can publish events according to their TD and listen for actions.

Services also have user permissions based on their assigned role, identical to other users. They can subscribe to events and publish actions.

Thirdly, services can provide service capabilities through request-response messages, that are restricted to certain roles. Users with the appropriate role can publish actions to the service. For example, an admin can publish actions to a management capability that cannot be used by operators. In this example the operator can publish to sources but not services.

To differentiate between access control of services from regular Sources, the Service use the 'service' prefix in their pub and sub addresses.
> svc.{serviceID}.{capabilityID}.*.>

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



