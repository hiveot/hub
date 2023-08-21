
## Mochi-co MQTT Authentication (planned)

Mochi MQTT supports hooks to perform authentication separate from authorization.

The Mochi adapter supports password, client certificate and JWT authentication. JWT tokens use H256 hashing for short lived token generation. Hashing protocol negotiation is not allowed [as this is considered a weakness](https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/). The hashing algorithm can be changed in future.

An asymmetric public/private key pair is used to generate the JWT token. Both ecdsa and ed25519 key pairs are supported.

On successful login a new short-lived token is issued. The lifetime of the token depends on the client type and is configurable. Refreshing the token is a simple matter of logging in again.

If the token has expired then a new token can be obtain using password login if a password is set, or must be issued separately. Devices will have to go through the provisioning process with out-of-band verification to obtain a new token.

Services are automatically issued a new token on restart by the launcher.
