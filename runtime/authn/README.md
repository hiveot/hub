# authn - HiveOT Authentication Service

## Objective

Manage Hub client profiles and corresponding authentication keys. 

## Status

This service is functional but breaking changes should be expected.

## Scope

In-scope is to provide identity management for users, devices, and services on the local network, with support for password, certificate, and token based authentication.

## Summary

This service provides the following capabilities:

1. Client management to add and remove Hub clients such as devices, users and services. Client information is persisted in an authentication store. 
2. Profile management for use by clients to update their profile and update login tokens.

Authentication can be used by the protocol bindings. It is up to each protocol binding to decide whether to use the authn service for generating session tokens. 

An optional golang client API is included and support for other languages is planned.

## Password Storage

Passwords are stored by the service using bcrypt or argon2id hashing. While argon2id is chosen as it is one of the strongest hash algorithms that is resistant to GPU cracking attacks and side channel attacks. [See wikipedia](https://en.wikipedia.org/wiki/Argon2). In future other algorithms can be supported as needed.

NATS uses bcrypt to storing passwords in memory. When using the NATS core passwords must be stored in bcrypt format so they can be applied to NATS.   

## Usage

This service is part of the hub runtime. 

To function as intended, the service must have access to the filesystem to store passwords and service authentication keys. The default path are the hiveot 'certs' directory for the certificate and hiveot 'stores/authn' directory for the profile and password storage. Location of files can be changed in the runtime.yaml config.

