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
   1. Authorization is provided through group membership
   2. Users subscribe to groups, not things
   3. Things and users are added to groups.
   4. Events published are send to the groups based on thing membership
5. Groups
   1. Each group has its own message store with retention policy
   2. Users can request to play back historical events
6. Directory
   1. The directory of Things is obtained by replaying the latest unique TD messages.
7. History
   1. The history is obtained by replaying the requested thing event for a given time range.

## Hub Client

Use of the Hub client is optional. The underlying messaging platform can be used directly using their native clients. For convenience, client libraries are provided in Golang and JS that abstract the underlying messaging platform to allow for drop-in replacements of messaging protocol. 

Each message platform plugin has a messaging definition for common tasks. Clients can also directly use the messaging platform.

## Support for Messaging Protocols

The main messaging protocol used by the Hub is NATS/JetStream. See the nats/README-nats.md for details on the subject definitions.

Support for mqtt can be added through the NATS mqtt bridge. 

Additional support for plain http, redis streams, rabbitmq, and others are considered for the future.


## Security

All connections require TLS to ensure encrypted communication.

IoT devices, Hub services and users use JWT tokens for authentication. The tokens are issued by the authn service. 

Web password based login is handled by via a REST service which uses the authn service to obtain a token.

Once authenticated, users use the token to access 'groups' and receive information from Things that are in the groups they are a member of. All communication goes through the nats server. There is never a direct connection to an IoT device by a user.

