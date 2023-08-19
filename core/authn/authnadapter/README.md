# Authn Service Adapters

The authn adapters apply authentication to the underlying messaging service. Currently a NATS-server and Mochi MQTT adapter are included.

## NATS Adapter

The NATS adapter configures the NATS server authentication. There is still work to do to separate it completely from the generic portion of the service.
The JWT tokenizer is not used in favor of using nkeys for client authn.

### NATS options

NKey and password identities are stored in NATS options. The bridge updates the  'users' and 'nkey' sections when devices, users and services are added or removed.


### NATS Server Token Generation

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


## Mochi-co MQTT Adapter (planned)

Mochi MQTT supports hooks to perform authentication separate from authorization. 

The Mochi adapter supports password, client certificate and JWT authentication. JWT tokens use H256 hashing for short lived token generation. Hashing protocol negotiation is not allowed [as this is considered a weakness](https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/). The hashing algorithm can be changed in future. 

An asymmetric public/private key pair is used to generate the JWT token. Both ecdsa and ed25519 key pairs are supported. 

On successful login a new short-lived token is issued. The lifetime of the token depends on the client type and is configurable. Refreshing the token is a simple matter of logging in again. 

If the token has expired then a new token can be obtain using password login if a password is set, or must be issued separately. Devices will have to go through the provisioning process with out-of-band verification to obtain a new token.

Services are automatically issued a new token on restart by the launcher.

