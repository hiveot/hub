# Hub Directory 

The Directory service manages the collection and querying of available Thing Description documents for use by consumers. It is part of the digital twin runtime.

## Status

This service is being updated to be part of the digital twin runtime. The main change is that it will operate in the runtime process instead of running as a stand-alone service. 


## Summary

The Directory holds the collection of TD documents that describe available Things. The TD documents are defined by the [W3C Web of Things (WoT) working group](https://www.w3.org/TR/wot-thing-description11/). TD documents are stored in their JSON format as described in the WoT specification. 

Compatible IoT devices update the directory through their agent. The agent publishes an event with the updated TD using one of the available protocol bindings. The runtime decodes the TD and updates its forms to include the hub's protocols for accessing properties, events and actions.  

The directory exposes three main API's: 
1. Event API to take TD events and update the stored TD document.
2. Action API to CRUD TD documents.
3. RPC API to list and query stored TDs.


## Storage

The directory service can utilize any store that implements the IBucketStore interface. 

Currently the embedded Key-Value store 'bucketstore/kvbtree' is used. 

## Build and Installation

This package is built as part of the hub's Digital Twin Runtime.  

See [hub's README.md](https://github.com/hiveot/hub/blob/main/README.md) for more details on building and installation.


## Usage

The directory can be queried through the hub CLI. The hub CLI calls the directory through the consumer hub-client. The consumer hub-client includes support for querying the directory. The directory can also be queried through rpc calls or action requests using one of the Hub's protocols. (tbd)

The Hub CLI command to list the directory content is:
> hubcli ld [agentID [thingID]] 

* agentID is an optional filter to only show the things published by that agent.
* thingID is an optional filter to show the details of a thing including its properties, events, and actions.

