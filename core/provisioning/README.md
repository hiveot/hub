# IoT Device Provisioning

## Objective

Provides a simple means to securely provision IoT devices using out-of-band device verification and
bulk provisioning. This is a golang implementation of
the '[idprov standard](https://github.com/wostzone/idprov-standard)'

Provisioned devices receive a client certificate that can be used to authenticate themselves
with the WoST Hub and its services.

## Roadmap

1. TBD Integrate with Hub discovery instead of stand-alone discovery. 

## Summary 

This project implements the 'idprov' IoT device provisioning protocol. It is intended for automated provisioning of IoT devices to enable secure authenticated connections from an IoT device to IoT services.

The typical use-case is that upon installing one or more IoT devices, the administrator collects the
device ID and corresponding out-of-band secret and provides these to the provisioning server using
the commandline utility or (future) web interface. When the devices are powered on the following
process takes place:

1. The IoT device discovers the provisioning endpoint on the local network using DNS-SD 
2. The IoT device requests a certificate from the provisioning server providing device identity and
   a hash of the out of band secret.
3. The provisioning server verifies the device identity by matching the device-ID and secret with
   the administrator provided information. If there is a match then the device is issued a signed
   identity certificate.
4. The certificate is then used by the device to authenticate itself with IoT service providers,
   publish its information, and receive actions and configuration updates.

The protocol uses the organizational unit (ou) field of the certificate to identify the client as an IoT device with corresponding permissions.


Preconditions:
* A certificate service must be available for generating certificates 


## Usage

On startup, the IDProv server publishes a DNS-SD record on the local network. IoT devices can
discovery it using the idprov client 'discover' function. Alternatively, IoT devices are provided
directly with the server address and port.

Once the idprov server is discovered, devices submit a provisioning request including their ID, out-of-band secret and public key. Once approved the server returns a certificate that is stored by the device and used for TLS connections with other Hub services. Periodically the Device renews the certificate by submitting a provisioning request halfway the existing certificate validity period.

In order to receive a certificate, the device ID and secret must be submitted out of band to the server before the devices requests provisioning. This can be done using the oob utility or via the Hub's admin UI if available. The admin UI shows a list of requests which the administrator can approve. Devices must retry repeatedly if their request returns the status 'waiting'.

If no special OOB secret is available, devices can use their MAC address as ID and serial number as its secret. This is up to the device itself. The easiest method for provisioning is the use of QR code or NFC tag on the device that can be scanned. A provisioning app can automatically pass this on as out-of-band verification to the server. For bulk provisioning a list of IDs and secrets can be provided using the oob utility.

The certificate provided in provisioning to the thing device must be used in order to connect securely to the hub gateway. All connections must use mutual authentication over TLS to obtain sufficient permissions.

## Installation

This service is included with the Hub as a core service and is bundled with the Hub installation.
