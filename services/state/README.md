# State Service

The state service provides a simple key-value storage service to store application and user state.


## Status

This service is functional but breaking changes can still be expected.

Future:
1. Limit the amount of data per client, nr records and record size 
2. Support for auto-expiry of keys.

## Summary

Devices, services and users can use this key-value store to persist basic state. It is intended for devices with no or limited storage to give them a place to persist their data and for web applications to store things like a dashboard configuration. As this service should be always available, it is part of the runtime core.

The state service uses the bucket store package that supports multiple implementations of the key-value store. The default store uses the built-in btree key-value store with a file based storage.  

The state service is intended for a relatively small amount of state data per client. Performance and memory consumption are good for at least 100K total records.

The state service is by default enabled in the runtime. It can be disabled through the runtime configuration. 

## Installation  

The service is embedded in the runtime and thus does not need installation.  

## Configuration

The service will run out of the box. The configuration can be modified in the runtime configuration. 


## Usage

The easiest way to use the state service client is through the provided state client.

Conceptual example of using the state to retrieve data. MyData can be a struct or native type.  
```go
  hc := ConnectToHub(...)
  stateClient := NewStateClient(hc)
  
  // write record
  myRecord := MyData{}
  stateClient.Put("mykey",&myRecord)
  
  // read record
  myRecord := MyData{}
  found,err := stateClient.Get("mykey",&myRecord)
```
