# authn service

## Objective

Provide authentication keys to clients on the local network. 

## Status

This service is alpha but functional.  


## Scope

In-scope is to provide identity management for clients on the local network. Login to the authn service will provide tokens required to authorize access to Thing resources.

## Summary

This core service manages HiveOT users and their credentials. It issues signed JWT tokens to authenticate with the NATS server.

Administrators manage users and passwords through the 'hubcli' commandline utility or through the authn management API. Creating users or resetting passwords requires an 'admin' key.

A note on nsc. Nats has a companion tool called 'nsc'. This tool is used to manage NATS operators, accounts and users, and authorization using the commandline.  It is separate from the nats server itself and interact with the server through the nats client.



#### **Password Storage**

Passwords are stored by the service using argon2id hashing. This is chosen as it is one of the strongest hash algorithms that is resistant to GPU cracking attacks and side channel attacks. [See wikipedia](https://en.wikipedia.org/wiki/Argon2). In future other algorithms can be supported as needed.

#### **Token Generation**

Authn uses JWT with H256 hashing for access and refresh token generation. Hashing protocol negotiation is not allowed [as this is considered a weakness](https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/). The hashing algorithm can be changed in future. 

An asymmetric public/private key pair is used to generate the access and refresh tokens. A public/private key pair is generated on startup. Services that verify access tokens must be given to the public key. Services on the Hub will look in the default location while services that run on a separate system should be provided the public key on installation. The service configuration enables the option to auto generate a new key pair on startup or to use the provided keys.

JWT is chosen as it includes a set of verified 'claims' which is needed to verify the userID. 

Clients request an access/refresh token-pair from the authn service endpoint by providing a login ID and a secret (password). The refresh token can be used to obtain a new set of access/refresh tokens. This approach allows for stateless verification of the access token for its (configurable) validity period with a default of 1 hour. The refresh token is valid for a longer time with a default of two weeks. When a refresh token has expired, the user has to login with credentials to obtain a new token pair. As long as the refresh token is renewed no login is required. Refresh tokens are stored in secure client cookies which means they are not accessible by javascript clients. 

Access and refresh tokens include claim with the IP address of the sender, which must match during verification. Any attempt to use the tokens with a different IP fails. The refresh token will be invalidated if an IP mismatch is detected even if it hasn't expired yet.

Hub services accept the access token within its validity period as long as they verify against the CA public key. The public key is available in the CA certificate or can be distributed as PEM file.

In short, services that accept access tokens perform stateless verification of the token against the CA public key and sender IP address. Clients must refresh their tokens if they receive an unauthorized response from one of the services and retry the request.

Weakness 1: Access to the tokens is the achilles heel of this approach. If a bad actor obtains an access token while it is still valid, and can spoof its IP address to that of the token, then security is compromised. This is somewhat mitigated by using TLS and requiring a valid server certificate, signed by the CA.


## Usage - old - still to be updated 


Code below is pseudocode and needs to be updated.

### Add user and set password

Using the authn CLI. This utility should only be accessible to admin users:
```bash
 bin/hubcli authn adduser {userID}      # this will prompt for a password
 
 bin/hubcli authn deleteuser {userID}

 bin/hubcli authn setpasswd             # this will prompt for a password
```

Using the service API:
```golang
  // an administrator client certificate is required for this operation
  authnAdmin := NewAuthnAdmin(address, port, clientCert)
  
  err := authnAdmin.addUser(userID, passwd)

  err := authnAdmin.setPasswd(userID, passwd)
```

### User login

```golang
  authnClient := NewAuthnClient(address, port)
  accessToken, err := authnClient.Login(username, password)
  // login sets the refresh token in a secure cookie. 
```

### User refresh auth tokens

```golang
  authnClient := NewAuthnClient(address, port)
  // The refresh token was stored in a secure cookie. 
  accessToken, err := authnClient.Refresh()
  if err != nil {
    // if token cannot be refreshed then request user login    
  }   
```

### Server validates token (authenticate)

Access tokens can be validated using the verification public key 

```golang
  pubkey := ReadPublicKey(pubKeyFile)
  authenticator := NewJwtAuthenticator(pubkey)
  claims, err := authenticator.VerifyToken(accessToken)
  if err != nil {
    return unauthorized
  }
```
