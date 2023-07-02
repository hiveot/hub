# Hub NATS Jetstream

HiveOT messaging configuration for NATS JetStream based messaging.


## Configuration

HiveOT runs out of the box with an embedded nats server configured as a nats leaf node. The server is run independently, configured for TLS, and a hiveot account. All further configuration takes place by the services using the account.

Configuration is handled in three places:
1. The static server configuration is defined in a nats-server.conf file that is generated on startup, or read if it already exists. It defines the listening address and port, TLS with certificate, memory limits, storage directory, and the hub account. This follows the nats-server configuration as documented by nats. 

2. The client configuration is handled by the Hub 'authn' service. It defines the clients, including devices, services and users and issues JWT tokens for authentication. The tokens constraints the allowed subjects. For example, 'things.>' for things, '>' for services, and 'groups.>' for users. The service itself subscribes to the 'things.authn.*.action' subject to receive requests for managing clients.  Only administrators are allowed to access this subject.

3. The stream configuration is handled by the Hub 'authz' service. It defines a stream per group including the client IDs that are members of that group. The allowed subjects depend on the role of the client in that group. The service itself subscribes to the 'things.authz.*.action' subject to receive requests for managing groups. Only administrators are allowed to access this subject.  

As account management is handled by the hub, the NATS nsc CLI tool should not be used for managing the account.

External server:
This setup allows the use of an external nats server. The core hub.yaml configuration should contain the external server address, port, account name and account private key file. 


Running as a leaf node implies the use the JWT decentralized authentication and authorization.  

NATS provide a CLI for the server and user configuration. The Hub is compatible with the CLI to be able to use the available tools.

It should also be possible to use an external NATS server if desired.

The Hub starts with the following steps:
1. Create the Hub CA certificate if it doesn't exist
2. Create/renew Hub admin client certificate if it doesn't exist
3. Create/renew Hub services client certificates or tokens if they doesn't exist
   - use launcher config or autoscan
4. Create the static server configuration if it doesn't exist
   - Hub nats account
   - default no subject access
   - add default users
     - admin user certificate
     - services certificates or tokens
3. create core service API subjects
    - authn subjects: services/authn/manage/{action} updateUser/deleteUser/...
    - authz subjects: services/authz/manage/{action} updateGroup/deleteGroup/...

The administrator can:
1. Manage users, devices, service authentication
   - CRUD users, devices
   - Create/renew jwt tokens
2. Manage group streams
   - CRUD streams
   - set subject permissions for streams clients



## Subjects

NATS supports subject based messaging, similar to topics in MQTT but using a '.' as separator and * > as wildcards.

Devices publish on, and subscribe to, the 'things' subject root.

| Devices          | Subject                                          |
|------------------|--------------------------------------------------|
| Publish Event    | pub things.{publisherID}.{thingID}.event.{name}  |
| Subscribe Action | sub things.{publisherID}.{thingID}.action.{name} |


Services use the 'service' subject root and a serviceID.

| Devices          | Subject                                |
|------------------|----------------------------------------|
| Publish Event    | pub service.{serviceID}.event.{name}   |
| Subscribe Action | sub service.{serviceID}.action.{name}  |


Users however are not allowed a direct subscription to things but instead subscribe to the group they are a member of. They will receive events from all the Things that are a member of the same group. 

The authorization plugin configures a NATS streams for each group. An ingress stream receives all events. Group streams subscribe to the ingress stream with a filter on Things that are a member of these groups. Users can subscribe to the groups they are a member of.


Users subscribe to groups:

| Users           | Subject                                                 |
|-----------------|---------------------------------------------------------|
| Subscribe Event | sub {group}.{publisherID}.{thingID}.event.{name}        |
| Publish Action  | pub {group}.{publisherID}.{thingID}.action.{name}       |
| Get TDs         | pub {group}.{publisherID}.{thingID}.action.td [1]       |
| Get History     | pub {group}.{publisherID}.{thingID}.action.history [2]  |


[1] publisherID and thingID are optional. Use '*' wildcard to filter on specific publisher or things.

[2] The payload defines the start date.
The subjects above will likely change. NATS supports message replay for latest and historical messages.  

[3] NATS supports request-response messages and inboxes. This can be used for provisioning, and token renewal. 



## Authentication 

NATS supports a variety of authentication methods. The Hub uses JWT tokens.


## Authorization

