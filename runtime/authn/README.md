# authn - HiveOT Authentication Service

## Objective

Manage Hub client profiles with authentication keys for the local network.

## Status

This service is being reworked for the Digital Twin Runtime. It is currently broken.

Implementation of custom roles is not yet complete. 

## Scope

In-scope is to provide identity management for users, devices, and services on the local network, with support for password, static key, and token based authentication.

## Summary

This service provides the following capabilities:

1. Client management to add and remove Hub clients such as devices, users and services. Client information is persisted in an authentication store. 
2. Profile management for use by clients to update their profile and update login tokens.

Authentication is used by the underlying protocol binding. It is up to each protocol binding to decide whether to use the authn service for generating session tokens. 

An optional golang client is included and support for other languages is planned.

## Password Storage

Passwords are stored by the service using bcrypt or argon2id hashing. While argon2id is chosen as it is one of the strongest hash algorithms that is resistant to GPU cracking attacks and side channel attacks. [See wikipedia](https://en.wikipedia.org/wiki/Argon2). In future other algorithms can be supported as needed.

NATS uses bcrypt to storing passwords in memory. When using the NATS core passwords must be stored in bcrypt format so they can be applied to NATS.   

## Usage

This service is included with the hub core. 

To function as intended, the service must store a CA certificate, server certificate and password file on the filesystem. The default path are the hiveot 'certs' directory for the certificate and hiveot 'stores/auth' directory for the profile and password storage. Location of files can be changed in the hub.yaml config.

