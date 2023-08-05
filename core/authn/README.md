# authn - HiveOT Authentication Service


## Objective

Manage Hub client profiles with authentication keys for the local network.

## Status

This core service is alpha but functional. Breaking changes can be expected.

Roadmap Items:
1. update documentation
2. integrate with authorization (NATS only)
3. startup selector for NATS and MQTT 

## Scope

In-scope is to provide identity management for clients on the local network.

The use of a password for login depends on the underlying messaging service. The server can use the Authn client to validate the password and obtain a token.


## Summary

This service provides the following capabilities:
1. Manage Hub clients such as IoT devices, services and users for use by managers. This provides CRUD capabilities for clients that is persisted in the file store.

2. Client authentication token creation and refresh for use by clients. This creates a new token and validates issued tokens.

The method of token handling depends on the underlying messaging service. The tokenizer generates new tokens when clients are added, and validates old tokens when they are refreshed.

The protocol binding uses the serializer 'ser' to serialize requests and responses. The default is JSON.

### NATS

The NATS specific portion consists of the tokenizer (see [AuthnNatsTokenizer.go](authnnats/AuthnNatsTokenizer.go)). 

NATS JWT tokens differ from plain old JWT in the following ways:
* token keys use es2519.
* use of signed nonce in the authentication process
* use of authorization subjects (topics) in the JWT token

This service receives action requests by subscribing to the subjects: 
* things.authn.manage.action.>
* things.authn.client.action.>

The subjects are created by the NATS version of the HubClient implementation.

When NATS tokens are issued they contain restrictions on the subject (topics) that can be published and subscribed to. By default this includes events, actions and an inbox, depending on the client type. Inbox prefix is used to prevent snooping of other clients inboxes.

Still TBD is the JetStream authorization and how this affects the token limits.

Tokens do not include limits for use with JetStream. Global limits for memory size and disk space are set centrally on the server.  

### MQTT

The MQTT server uses the [built-in JWT tokenizer](authnservice/AuthnServiceTokenizer.go). 

## Usage
This service is included and launcher by the Hub core.

To function as intended, the service must store a CA certificate, server certificate and password file on the filesystem. The default path are the hiveot 'certs' directory for the certificate and hiveot 'stores/authn' directory for the profile and password storage.

