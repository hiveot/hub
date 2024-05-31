# IdProv - IoT Device Provisioning

## Objective

Provides a simple means to securely provision IoT devices using out-of-band device verification and bulk provisioning. 

Provisioned devices receive a device token that can be used to authenticate themselves with the Hub.

## Status

This service is being converted to work with the HiveOT digital twin runtime. Breaking changes can still be expected.

TODO:
1. ~~run as external service for the new runtime~~
1. Add monitoring for rogue DNS-SD publications 
2. TBD: Checks if multiple connections are made by the same device using different device IDs
3. TBD: flag suspect devices
4. TBD: MAC address check during provisioning

## Summary 

The purpose of this service is to enable fast and easy onboarding of one or multiple IoT devices without the need to setup or configure the device itself. 

The typical use-case is that upon installation of one or more IoT devices, the administrator collects the device ID and corresponding out-of-band secret and provides these to the provisioning server using the commandline utility or Hub management web interface. When the devices are powered on they auto-provision following these steps:

1. The administrator uploads the pre-approved list of device IDs with their public key as provided by the factory. A MAC address can be used instead of a public key, in which case the device's MAC will be matched and the device's public key recorded.
1. The IoT device discovers the provisioning service on the local network using DNS-SD. 
1. The IoT device connects to the server on the provisioning port and initiates the start of the provisioning process, providing its actual ID, public key and MAC address.
1. The server verifies the ID, public key and MAC address. 
   A: if client ID is unknown then the administrator must approve the request. The server returns a 'pending approval' answer.
   B: if the client ID and public key are known, the client is added and a short lived auth token is returned. 
   C: if the client ID and MAC are known, the client is added along with the provided public key and a short-lived auth token is returned.
5. After receiving the token, the client uses it to connect to the Hub with its own ID. As the token is short lived, the device must immediately refresh the token from the auth service. The lifespan of tokens issued by the auth service is configurable and defaults to 1 month. 
6. Periodically the device must check the remaining token lifespan and requests a new token from the auth service when less than half its lifespan remains.  The lifespan can be determined by decoding the JWT token and verifying the expiry date. 

Devices that are pre-loaded with the server CA certificate, are secured against a rogue server as establishing a connection with this server will only succeed with the valid server.

Devices that do not possess the server CA certificate are subject to a man-in-the-middle attack that relays the provisioning request to the actual service to obtain a device token.
To mitigate the risk of issuing tokens to rogue agents, the Hub:
1. monitors the network for DNS-SD records of provisioning servers. If a rogue provisioning server is detected it notifies the administrator and disables the provisioning process until the administrator re-enables it. (todo)
2. Checks if multiple provisioning requests come from the same device. Flag the device for review. (todo)
3. Checks if multiple connections are made by the same device using different device IDs. Flag the devices for review. (todo)
4. Checks if the auth token matches the client MAC address provided during provisioning. Flag the device for review. (todo)  

Administrators can further reduce the risk by:
1. Preload the devices public keys into the service. Only known devices are issued a token. A rogue agent can't use the issued token as it doesn't know the private key.  This is the preferred method for larger installations.
2. Verify the list of provisioned devices afterwards. Devices that are not showing up might have been taken 'hostage' by a rogue agent. 
3. Test each device functions as expected. If a rogue agent obtained a token the device might not show up, unless the agent remains active as a relay.

The highest risk exists when a server CA is not installed on the devices, the device ID and public key are not pre-loaded in the server, and a rogue provisioning server remains undetected by the real provisioning service. In this case a rogue server can receive a provisioning request and forward it to the real service with its own public key. The administrator might approve it as the timing is right. 


## Usage

On startup, the IDProv server publishes a DNS-SD record on the local network. IoT devices can discovery it using the idprov client 'discover' function. Alternatively, IoT devices are provided directly with the server address and port. The default port is 8445.

Once the idprov server is discovered, devices submit a provisioning request including their ID, out-of-band secret and public key. Once approved the server returns an auth token and server URL that are stored by the device. Periodically the Device renews the token using the auth services midway the token validity period.

In order to receive a token, the device ID and public key or MAC address must be submitted out of band to the server before the devices requests provisioning. This can be done using the oob utility or via the Hub's admin UI if available. The admin UI shows a list of requests which the administrator can approve. Devices must retry repeatedly if their request returns the status 'pending'.

## Installation

This service is included with the Hub as a core service and is bundled with the Hub installation.

It will run out of the box without any configuration when activated using the launcher.

