# Certs Service

## Objective

Manage certificates for authentication of hub services, IoT devices and consumers.

## Status
This service is functional and can issue IoT device certificates, service certificates and end-user certificates.

For consideration:
- Support for lets encrypt to obtain service certificates

## Summary

This service manages authentication certificates for use by services, IoT devices and end-users on the local network:

1. A self-signed Hub CA certificate for generating signed certificates. This is usually done by the administrator using the CLI or the admin web console if such is included.
2. Hub service certificates for each Hub service
3. IoT device client certificates for each IoT device.
4. User client certificates for end-users 

This service currently uses self-signed certificates and has no external dependencies. Support for lets-encrypt and 3rd party certificate import are possible for the future.

## Certificate Revocation

When certificates are renewed the old certificate must no longer be used. Certificate revocation checks are currently not implemented but might be added in the future. 


## Hub CA Certificate

By default, the Hub utilizes a self-signed CA certificate, created during first install. Use of a 3rd party generated signing certificate is typically not necessary as the Hub is only using these certificates on the local network to identify local IoT devices and users.

The CA certificate has a default 2 year validity period. A new CA certificate is generated every year using the same public/private key pair. The new CA certificate then has a year to be automatically distributed to the IoT devices when they renew their own certificate. 

It is recommended to import the CA certificate in the browser to avoid unnecessary warnings when accessing the Hub. The alternative is to accept the browser warning. 

## Hub Service Certificates

Hub services can request a new certificate for use with TCP connections. This certificate is signed by the CA. It is intended for authentication of the service to its clients and as a client to authenticate to other services. 

Service certificates are only used with TLS connections over TCP. UDS connections do not use TLS. UDS connections are secured through their socket file permissions. MiM attacks are not possible without root access to the device, in which case all security has already been broken.

A service certificate is an organization certificate and not a domain certificate. They are only intended to be used on the local network (domain is local). The certificate CN contains the service ID, its OU identifies it as a Hub service and SAN names contain both the hostname and the ip address of the device the service is running on.

Service certificates have a short validity period. They are renewed by the service on each service restart.

It is recommended to NOT expose any service to the internet unless a proxy server is used. The proxy server can use a certificate from Lets Encrypt.

Proposal: bridge services connect to each other over the internet. They are effectively a proxy between Hubs and utilize a - preferably Lets Encrypt - certificate. Bridge services use a manual adoption process where the administrator adds the remote bridge address and accept or reject the offered certificate. Certificate renewal will re-use the keys used to generate the certificate and does not require redistribution. 


## IoT Device Certificates

IoT devices that register themselves with the Hub to publish 'Things' receive a client certificate upon registration. This certificate is unique to the device and signed by the CA. 

The provisioning service can be used to automatically provision IoT devices using out-of-band secrets. The administrator can upload a batch of device ID's and corresponding secrets. The device can request a certificate by providing its ID and secret. If the secret is recognized, the certificate is issued. 

Provisioning secrets are optional. Requests without a secret will be stored and can be manually approved by the administrator.

The default device certificates has a validity period of 30 days, before which they can be automatically renewed using the provisioning service. Certificates can also be installed manually onto devices that support it. In this case any validity period can be used when generating the certificate. 

The certificate CN contains the device ID, while its OU identifies it as an IoT Device. The device ID is its unique identifier for the local network.  

IoT device certificates are used by the Hub to authenticate the device. This allows the device to publish the TD documents of its 'Things', publish events and receive actions. Security only allows for a device to publish/subscribe for Things of which it is the publisher.


## User Authentication Using Certificates

Users can be issued a client certificate so they don't have to login using the login name and password. The certificate CN contains the user login name while the OU identifies it as a User.

Consumer client certificates have a validity of 1 year, before they must be renewed.  

The Hub CLI provide a command to create user certificates manually. See the CLI for more information.

## Build and Installation

This package is built as part of the Hub. 

See [hub's README.md](https://github.com/hiveot/hub/blob/main/README.md) for more details on building and installation.

### Configuration

When launched from the installation folder this service works out of the box without requiring configuration.

The service has its configuration in the application 'config' directory. This is either the  './config' subdirectory of the application installation directory, or the /etc/hiveot/config directory as described in the hub README.md

The following files are used:
* config/certs.yaml        - Service configuration
* certs/caCert.pem  - CA public certificate in PEM format
* certs/caKey.pem   - CA private key. Read-only for hub process only.

Issued service certificates are stored:
* certs/<service>Cert.pem - Issued service public certificate in PEM format
* certs/<service>Key.pem  - Issued service public/private key in PEM format

IoT device certificates and user certificates are not stored on the Hub.

## Usage

This service is intended to be started by the launcher. For testing purposes a manual startup is also possible. In this case the configuration file can be specified using the -c commandline option. 

The Hub CLI provides a commandline interface for managing certificates. For more information see:
> hubcli certs -h

The service API is defined with capnproto IDL at:
> github.com/hiveot/hub/api/capnp/hubapi/certs.capnp

A golang interface at:
> github.com/hiveot/hub/pkg/certs/ICerts

### Golang Clients

An easy to use golang POGS (plain old Golang Struct) client can be found at:
> github.com/hiveot/hub/pkg/certs/capnpclient

To use this client, first obtain the capability (see below) and create the POGS instance. For example, to create a device certificate the provisioning service does this:
```
  deviceCap := GetCapability()                       // from an authorized source
  deviceCert := NewDeviceCertsCapnpClient(deviceCap) // returns IDeviceCerts
  certPEM, caCertPEM, err := deviceCert.CreateDeviceCert(ctx, deviceID, pubKeyPEM, validityDays) 
```

where GetCapability() provides the authorized capability, and NewDeviceCertsCapnpClient provides an the POGS implementation with the IDeviceCerts interface.

To obtain the capability to use the ICerts apis, proper authentication and authorization must be obtained first. In this case, the certs package is intended for internal use by the CLI and select services such as the provisioning service. These leverage a bootstrap mechanism such as the socket  the service is listening on.
