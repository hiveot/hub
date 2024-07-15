# Hub NATS Jetstream

HiveOT configuration for NATS JetStream based messaging.

## Status - NOT FUNCTIONAL

This transport is being reworked for use with the HiveOT runtime. It is currently broken.

The stuff below is old

## Configuration - OLD

HiveOT runs out of the box with an embedded nats messaging and streaming server. This server is configured to runs with TLS over TCP and WebSocket on the default NATS ports.

NATS server configuration and startup is handled through following steps:

1. Read the general hub.yaml configuration. 
   * The server section contains the server address, port, certificates and storage directory.
   * The authn section contains the authentication settings.
   * The authz section contains the authorization settings.
   
2. Generate the static server configuration.
   1. Obtain server address, port, and account from hub.yaml
   2. Obtain the users from authn
   3. Obtain the stream config from authz, including authorization

3. Start the embedded server with the server configuration file. 

4. Start the authn service. 
   * Subscribe to authn action subject on things.authn.*.action.>
   * On adding/removing users, update the nkeys section of the server config and reload

5. Start the authz service.
   * Add pub/sub permissions to each member of each group stream 
   * Reload the server
   * Subscribe to authz action subject on things.authz.*.action.>
   * On membership change, update the authorization section of the server config and reload

## Authentication Using JWT, NKeys or Callouts

NATS supports various authentication methods, but they cannot be used together. A choice must be made. Below issues around NKeys and JWT:
1. JWT combines authentication and authorization. Changes to permissions requires that the client obtains a new JWT token. This in turn requires a mechanism to revoke old tokens and for the client to renew their JWT token so the new permissions take effect. This adds a lot of extra management to ensure quick token updates. There is also the security risk that reducing permissions doesn't take effect as long as the client uses the old token unless it is revoked. 
2. NATS doesn't support user authorization separate from JWT. There is no option to allow or deny subjects outside the JWT token as the users authorization in the server config cannot be used together with JWT.
3. NATS JWT's main advantage is that it provides decentralized management of users. The server doesn't need to know each user. If the application requires tracking users then the JWT advantage doesn't apply and becomes a disadvantage (see authorization critique above).  
4. Nats JWT cannot be used with other authentication methods. Using JWT disallows the use of username/password authentication by web clients. As a result, a separate HTTP webserver is needed to allow obtaining a token through password login. Why can't both be used?
5. NKeys don't have expiry built-in as they are simply asymmetric key pairs. The support key expiry a separate process must be added to track validity and remove expired keys. 
6. Nkeys don't carry claims. An extra request to an authn service is needed to obtain extra info such as client type (service, device, user) and display name.
7. NKeys can be used together with password authentication avoiding the need for a separate REST API to login.
8. NKey also work with the 'unauthenticated' user account. This allows unauthenticated users to connect to request a key after password login and authenticate using the key. With JWT this isnt possible.
9. Callouts while allowing to run your own authentication mechanism, does not allow the handler to verify NKeys or JWT keys used to authenticate. You have to re-write token verification.  
10. Callout has to use the application account and cannot run in a separate account. This blocks use best practices of separating authn service from the application.
11. Callout configuration is confusing and not well documented (yet) and changes can still occurr. The configuration of the Issuer, Account and IssuerAccount differbetween server and operator mode as is the audience field of the issued token (nats-server/issues/4335).

Ideally JWT should be usable for just authentication together with NKey and password authentication. Server side authorization should be usable with JWT as it is with NKeys and Passwords authenticated clients. The pubsub limits claims in JWT tokens is bonus and should not be required. This allows the use of JWT for short lived tokens. 

For HiveOT it is clear that using JWT is not really an option but that use of NKeys is feasible.

## Subjects

NATS uses subject based messaging, similar to topics in MQTT but using a '.' as separator and * > as wildcards.

Devices and services publish on, and subscribe to, the 'things' subject root.

| Devices          | Subject                                          |
|------------------|--------------------------------------------------|
| Publish Event    | pub things.{bindingID}.{thingID}.event.{name}    |
| Subscribe Action | sub things.{bindingID}.{thingID}.action.{name}.* |

Where {bindingID} is the ID of the device or service publisher, and {thingID} is the sub-device or service-capability being addressed.

Like users, services can also subscribe to events and publish actions. 

Users are not allowed a direct subscription to things but instead read events from the group they are a member of through NATS JS consumers.  

The authorization plugin configures a NATS streams for each group. An ingress stream receives all events. Group streams use the ingress stream as a source with a filter on subject of Things that are a member of these groups. Users can read messages from the groups they are a member of.

Devices publish events that follow these steps in their flow:
> device -> $events stream -> group stream -> consumer -> user

Where multiple group streams can use the same event subject as data source.  

