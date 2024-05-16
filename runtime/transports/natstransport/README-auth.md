# Authn - NATS authentication and Authorization


### NATS options

NKey and password identities are stored in NATS options. Nats auth updates both the  'users' and 'nkey' sections when devices, users and services are loaded.

Since authorization is tightly coupled with identity in NATS config, the server will need both in order to update a user, device or service.


### nkeys - NATS Server Token Generation

NATS server uses ed22519 based asymmetric public/private key pairs for 'nkeys'. The private key stays with the client and the public key is shared with the server. The server sends a 'nonce' challenge on connect that is signed by the client using its private key. The resulting signature is verified by the server using the public key on record.

When using NATS server with nkeys there is no token generation. In this case the adapter uses public key as the token. This still gives a good protection as the private key is required to sign a 'nonce challenge' on login. However, these tokens have no expiry built-in. The client can update the key pair and update the public key periodically, after which the old public key is removed from the server.  

NATS JWT is not used as it disables password, certificate, and nkey authentication. In addition, it requires that authorization is stored in the authentication key. This is bad practice as this leaks client information and requires updating the key on the client after permissions changes.

External authentication (callouts) is not officially released and suffers from poor documentation and inconsistent configuration between server and operator configuration. It is also not clear where authorization takes place when using callouts. It looks like the callout handler must include this in an internally generated JWT token but this isn't clear.  There is potential here once callouts are released and is stable. 

The bottom line is that configuration of authorization should remain server side.

### NATS Message Subject

NATS uses subject to address messages (similar to topics in mqtt). The authn service receives action requests by subscribing to the subjects:
* things.authn.manage.action.>        // manage capability 
* things.authn.client.action.>        // client capability

These subjects are created by the NATS version of the HubClient implementation.

### NATS nsc

A note on nsc. Nats has a companion tool called 'nsc'. This tool is used to manage NATS operators, accounts and users, and authorization using the commandline.  It is separate from the nats server itself.

As nsc has its own storage of users and accounts it is currently not compatible with hiveot.  

Support for external NATS server is a future consideration. It would require a manual setup of the application account and invoking the nsc tool over the commandline to manage users and authorization.

## authz

### Subscribe to Groups (streams)

Groups are simulated in nats using streams and ephemeral consumers.

Bindings publish Thing events using the subject format:
> things.{bindingID}.{thingID}.event.{eventType}.{instance}

These events are captured in a central '$events' stream.

When a group is created by the administrator, the associated stream is created using the group name.

When a Thing is added to the group by the group manager, its subject is added as a stream source using $events as the source stream.

A group streams have a source defined for each Thing that is a member of the group. Each source is the combination of $events stream and the subject of the events captured in the stream.

To access a group stream, the user creates an ephemeral consumer for the stream. Under the hood this uses the subject $JS.API.CONSUMER.CREATE.{groupName}. Only members of the group can publish to this subject. Similar for DELETE, INFO.

To read from a group stream, users also need permission for $JS.API.CONSUMER.MSG.NEXT.{groupName}.>


The authz service has a listGroups action to allow a client to list the groups they are a member of.

In summary, the above setup accomplishes the following:
1. Authorization to receive events only for Things that are in the same group(s) as the user
2. Archiving of events for each group with its own retention period
3. Users can retrieve the latest value of each event
   4Users can retrieve historical events
   5Users only need to subscribe to a group to receive relevant events. No need to subscribe to individual things.


### Publish Actions

Users can publish actions by writing them to the group stream on subject:
> things.{bindingID}.{thingID}.action.{actionID}.{clientID}

TBD: this requires that the published subject are constrained to to those defined with the stream.

The $actions stream has a source for each group stream using subject "things.*.*.action.>".

Bindings subscribe to the $actions stream using a durable consumer to receive requests. If multiple actions with the same ID are received after a reconnect, then only the last one should be applied by the binding.

