# HiveOT Core  

The primary objective of the HiveOT core is to support publishing and subscribing events and actions. The payloads are based on the W3C WoT standard. Authorization is provided through group membership.

## Capabilities

The HiveOT core consists of a thin abstraction layer on top of a messaging platform. It provides the following capabilities:

1. For Devices
   1. Publish events on behalf of their Things 
   2. Subscribe to actions on behalf of their Things
   3. Authenticate with tokens
   4. Obtain tokens based on authentication policy
      1. Default open:
         1. Local network devices can obtain tokens freely 
      1. Default closed: 
         1. Device must provide a signed token to publish events
         2. Tokens can be obtained through provision using out of band secrets
         3. Tokens can be obtained in a timed window (user driven)
2. For Users
   1. Login with password or token
   1. Obtain new refreshed tokens after login
   1. Subscribe to groups they are a member of
   2. Have an operator or viewer role in a group
   1. Can publish actions to Things in their group if they have an operator role
   2. Each user account is also its own group 
3. Services/Applications can act as a device and user
4. Authorization 
   1. Authorization is managed through group membership
   2. Users subscribe to groups, not things
   3. Things and users are added to groups.
   4. Events published are send to the groups based on thing membership
5. Groups
   1. Each group has its own message store with retention policy
   2. Users can request to play back historical events
6. Directory
   1. The directory of Things is obtained by replaying the latest unique TD events.
7. History
   1. The history is obtained by replaying the requested thing event for a given time range.

## Hub Client

Use of the Hub client is optional. The underlying messaging platform can be used directly using their native clients. For convenience, client libraries are provided in Golang and JS that abstract the underlying messaging platform to allow for drop-in replacements of messaging protocol. 

Each message platform plugin has a messaging definition for common tasks. Clients can also directly use the messaging platform.

## Support for Messaging Protocols

The Hub's core is implemented for multiple messaging servers:
- NATS/Jetstream embedded server using NKeys and passwords for authentication
- NATS/Jetstream embedded server using callouts for JWT/password authentication
- MQTT using mochi-co embedded server using JWT/password authentication

Support for other messaging servers can be added. Only a single messaging core can be used at the same time.

Additional support for plain http, redis streams, rabbitmq, and others are considered for the future.


## Authentication

All connections require TLS to ensure encrypted communication.

IoT devices, Hub services and users use keys, tokens or passwords for authentication. The tokens are issued by the authn service. 


## Authorization

Authorization allows clients to receive events and publish actions based on their group roles. A group consists of IoT Things as source and Users as consumers. Services can act as both Thing or User.

IoT devices publish their Things on the 'things' address. Authz ensures that these addresses are mapped to the group address.

The group address format is: {groupID}.{publisherID}.{thingID}.{messageType}.{name},
where '.' is the separator for nats, or '/' is the separator for mqtt.

All Group members have subscribe permissions to all events in the group. Operators and managers can also publish actions to the group.

The method of mapping thing events to groups depends on the underlying messaging service used.

## Services

Authorization to use services depend on the service type.
Services use the serviceID as the publisherID and a service capability as ThingID.



## NATS

Nats groups are implemented using streams. When subscribing to a group a consumer is created for that group. Group members must have permissions to create a consumer and read messages.
* Pub: $JS.API.CONSUMER.CREATE.{groupID}
* Pub: $JS.API.CONSUMER.LIST.{groupID}
* Pub: $JS.API.CONSUMER.INFO.{groupID}

To receive events:
* Sub {groupID}.*.*.event.*

To send actions:
* Pub {groupID}.*.*.action.*.{senderID}  (where senderID is the clientID)

The stream has a single source, being: {groupID}.>

