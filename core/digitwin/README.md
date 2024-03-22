# DigiTwin Service

## Status

The DigiTwin service is in the planning phase. When completed, it replaces the directory service and the 'latest' capability (API) of the history service. 

## Introduction

The IoT Digital Twin collection is the hearth of the HiveOT Hub. The DigiTwin service keeps a digital representation of all IoT devices that are registered with it. 

Each digital twin consists of a device's TD document and a state object holding the properties, latest event values and the queued actions that have not yet been delivered.

The objectives of this service are:
1. Keep a digital twin instance for each registered IoT device.
2. Persist the digital twin collection between restarts.
3. Allow IoT devices to register their digital twin with their values.
4. Allow clients to query the twins by Thing ID, Agent, and device type.
5. Notify clients of changes to property values and events.
6. Support various communication protocols through protocol bindings.
7. Hold device configuration if needed.

IoT devices update the TD document and state of their digital counterpart by communicating on their separate communication channel with the DigiTwin service. They retrieve action requests that are provided by the DigiTwin service.  

Users of IoT devices communicate with the DigiTwin service to receive information and send action requests. They do not connect or communicate directly with the IoT devices themselves. Instead they operate on the digital twin of the devices. 

This approach provides near complete isolation between IoT devices and their users, which greatly enhances security. At no time is there a direct connection between user and IoT device.

Action requests from users are sent to the DigiTwin until they can be handed off to the IoT device or expire. This allows intermittently connected devices to receive action requests that were sent while they were offline. 

## IoT Device Connections

IoT devices can connect directly to the Hub DigiTwin service through one of the protocols, and register their TD document, upload their property values, send events and receive action requests.
For devices that cannot persist their own configuration the service can also provide the last known configuration values when a device starts up.

## Protocols

The HiveOT Digital Twin service supports multiple communication protocols.

The primary protocol is a connection based message bus, either MQTT or NATS. This supports pushing messages from the Digital Twin to the client once connection is established.

The second protocol is Websocket based, which utilizes a permanent connection and also supports pushing messages from the Digital Twin to the client.

The third protocol is REST over HTTPS. This connectionless interface supports pushing data to the digital twin and querying for data. It does not however, support data push from the Hub to the device or user client.

The fourth protocol is server-side-events (SSE) which complements REST over HTTPS with server push.

## Persistance

This service persists the TD documents it receives from IoT devices and services, aka 'Things'. On shutdown and startup it restores the TD documents and creates a digital twin for each of the 'active' Things. Active Things are Things that have send data to the service in recent time. Inactive Things can be unloaded to free up memory.

A digital twin also includes the state of the IoT device and constantly synchronizes this state with the actual IoT device. Synchronization takes place using event and action messages via one of the supported protocols.

Changes to this state is periodically persisted, and stored on shutdown. The storage engine uses the 'bucket-service' API which is implemented using one of the embedded databases: pebble, kvbtree, or bolts.  On startup the state of active things is loaded from the database.

When querying the state of a Thing, the state of the digital twin is returned. If the Thing isn't yet active it will be activated and its state is loaded into memory. 

## Authentication and Authorization

The authentication mechanism depends on the protocol used. 