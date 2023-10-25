# State Service

The state service provides a simple key-value storage service to store application and user state.


## Status

This service is currently in development.


## Summary

Applications and services that need to persist state can use this key-value store. This is also available to consumers for storing state of their web application, for example a dashboard layout.

The state service uses the bucket store package that supports multiple implementations of the key-value store. The default store uses the built-in btree key-value store with a file based storage.  

The state service is intended for a relatively small amount of state data. Performance and memory consumption are good for at least 100K total records. 

## Installation 

The service is installed in the plugin directory. The hiveot Makefile includes the build and install commands similar to other core services. 

The state service is intended to be started by the launcher and is included in the default launcher.yaml configuration. For testing purposes a manual startup is also possible. 

## Configuration

No configuration is needed. The service will run out of the box.


## Usage

The service API messages are defined in the stateapi directory. Data can be retrieved and stored using the rpcrequest message of the hubclient. A native API is supported using the stateclient/StateClient implementation.

Conceptual example of using the state to retrieve data. MyData can be a struct or native type.  
```go
  hc := ConnectToHub(...)
  stateClient := NewStateClient("",hc)
  
  // write record
  myRecord := MyData{}
  stateClient.Put("mykey",&myRecord)
  
  // read record
  myRecord := MyData{}
  found,err := stateClient.Get("mykey",&myRecord)
```
