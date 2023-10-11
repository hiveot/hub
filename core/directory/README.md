# Hub Directory 

## Objective

Manage the collection of available Thing Description documents for use by consumers.

## Status

This service is functional but breaking changes can still be expected.


## Summary

The Directory holds the collection of TD documents that describe available Things. The TD documents are defined by the [W3C Web of Things (WoT) working group](https://www.w3.org/TR/wot-thing-description11/). TD documents are stored in their JSON format as described in the WoT specification. 

Compatible IoT devices can directly publish their TD to the Directory through the capability obtained from the Hub gateway. 

Incompatible IoT devices will need a protocol binding service that creates or obtains a TD document for the device Things (an IoT device can have multiple Things). This protocol binding becomes the publisher for all Things under its management. 

The primary tasks of this service is to:
1. Store TD documents submitted by IoT devices
2. List and Query TD documents requested by consumers


## Storage

The directory service can utilize any store that implements the IDirectory interface. 

Currently a built-in in-memory Key-Value store 'directorykvstore' is used that periodically writes its contents to disk. The file is located in config/directorystore.json.

Additional storage options are planned such as mongodb, sqlite.

## Build and Installation

This package is built as part of the Hub.

See [hub's README.md](https://github.com/hiveot/hub/blob/main/README.md) for more details on building and installation.



### Configuration

When launched from the installation folder this service works out of the box without requiring configuration.

The service has its configuration in the application 'config' directory. This is either the  './config' subdirectory of the application installation directory, or the /etc/hiveot/config directory as described in the hub README.md

The following configuration files are used:
* config/directory.yaml        - Service configuration
* config/directorystore.json   - Directory storage file  


## Usage

This service is intended to be started by the launcher. For testing purposes a manual startup is also possible. In this case the configuration file can be specified using the -c commandline option.

The Hub CLI provides a commandline interface for viewing the directory. For more information see:
> hubcli directory -h


The service API is defined with capnproto IDL at:
> github.com/hiveot/hub/api/capnp/hubapi/directory.capnp

A golang interface at:
> github.com/hiveot/hub/pkg/directory/IDirectory

### Golang POGS Client

An easy to use golang POGS (plain old Golang Struct) client can be found at:
> github.com/hiveot/hub/pkg/directory/capnpclient

To use this client, first obtain the capability (see below) and create the POGS instance. For example, to list the directory:
```golang
  directoryCap := GetCapability()                       // from an authorized source
  readAPI := NewReadDirectoryCapnpClient(directoryCap)   // returns IReadDirectory
  jsonDocs, err := readAPI.ListTDs(ctx, limit, offset) 
```

where GetCapability() provides the authorized capability, and NewReadDirectoryCapnpClient provides an the POGS implementation with the IReadDirectory interface.

To obtain the capability to use the IDirectory apis, proper authentication and authorization must be obtained first. The gateway service provides this capability on successful login.
