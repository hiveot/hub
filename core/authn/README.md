# authn - HiveOT Authentication Service

## Objective

Manage Hub client profiles with authentication keys for the local network.

## Status

This core service is alpha but functional. Breaking changes can be expected.

Roadmap Items:
1. update documentation
2. integrate with authorization service as needed
3. startup selector for NATS and MQTT 

## Scope

In-scope is to provide identity management for users, devices, and services on the local network, with support for password, static key, and token based authentication.

## Summary

This service provides the following capabilities:

1. Management capability to add and remove Hub sources and users. Client information is persisted in an authentication store. Storage is a plugin that implements the storage API.

2. Client capability for token creation and refresh. This is intended for clients to refresh authentication tokens and update their profile.

Authentication itself depends on the underlying messaging server and is provided through a so-called adapter. A Nats and mochi-mqtt adapter are included.

An optional golang client for both management and client capability is included and support for other languages is planned.

## Password Storage

Passwords are stored by the service using bcrypt or argon2id hashing. While argon2id is chosen as it is one of the strongest hash algorithms that is resistant to GPU cracking attacks and side channel attacks. [See wikipedia](https://en.wikipedia.org/wiki/Argon2). In future other algorithms can be supported as needed.

NATS uses bcrypt to storing passwords in memory. To support this, it might be required to storage password in this format so it can be applied to NATS.  

## Usage
This service is included and launcher by the Hub core.

To function as intended, the service must store a CA certificate, server certificate and password file on the filesystem. The default path are the hiveot 'certs' directory for the certificate and hiveot 'stores/authn' directory for the profile and password storage.

