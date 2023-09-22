# Streams and Channels

## Introduction

Much of IoT data collected from devices comes in the form of events. Similarly, controlling IoT devices is done through 'actions'. The W3C WoT group has recognized this and devotes a large portion of the TD document standard around this. 

HiveOT in its previous incarnations has followed a similar line of thought and is now committed to (a subset of) the WoT standard. Specifically, the TD document describing IoT devices, aka "Things". 

Implementing support for the WoT however is not a simple matter. There are many considerations as described in ['challenges'](challenges.md).

The 'Streams and Channels' iteration of hiveot is trying to learn from the previous iterations and address these challenges. This document describes the concepts of this implementation.

Why even consider this? Simplicity on multiple levels:
* The API surface is very small. Open/close a channel and read/write a stream of data.
* Conceptually, channels are easy to understand. A stream is like water while a channel is the plumbing to direct the water.
* Streams can have a high throughput even when latency is high.
* Streams can be stored and replayed to extract data.
* Streams can be forked and routed. For example, routing a notification stream to one or multiple input channels.
* A channel can filter a stream in real-time. For example, a notification channel can filter out only notifications of interest.


## What Are Streams and Channels

What exactly do you mean with streams and channels?

A stream contains IoT data and media. A channel passes the stream between two endpoints. As an analogy, streams is like water and channels is like plumbing.

A channel:
1. has a description document needed to open a channel  
2. has a data type: event, action, audio, or video
3. connects a source endpoint to a sink endpoint
4. can use any secure compatible transport, such as a unix pipe, udp/tcp connection, webrtc.
5. transports a stream from a source endpoint to sink endpoint
6. support multiple stream encoders/decoders for data, audio or video, depending on the channel type and limitations of the underlying transport.   
7. waits for the sink endpoint to be ready before transporting the stream
8. is unidirectional 
9. event and action channels each carry a specific message data type.
10. channels require authentication to open
11. channels require authorization to write
12. channels can transform the stream data format if a compatible transformer is available
13. channels can filter the stream  
 
Channels are defined by a CDD, channel description document, describing:
* channel ID
* source endpoint ID (optional)
* destination endpoint ID (optional)
* stream type: event, action, audio, video, custom
* codecs: json, h265, mpeg, webrtc
* transports: unix, tcp, tls, mqtt, webrtc
* ca: CA certificate used to verify connections
* filter: optional stream filter 

## Establishing a Channel

Channels use signalling in order to negotiate a transport and encoder compatible with the stream type. The CDD can be predefined, passed out of band, or send over the transport ahead of the stream. This depends on the transport used. WebRTC has signally support so that is used. 


## Opening A Channel

Opening a channel involves 3 parties. The initiator, the mediator and the accepting party. The initiator can be a source endpoint that sends the stream or a sink endpoint the wants to receive a stream. The initiator or accepting party can also act as the mediator. 

In order to open a channel both endpoints must be connected to the mediator.

1. Connect to the mediator if not already connected
2. Send a hail to the accepting party with the CDD and authentication token 
3. Receive a response.
   1. If the endpoint is incompatible the request is rejected with the 'incompatible stream' reason.
   1. If the endpoint is compatible, the accepting party has to accept the request. This can require confirmation by a user.
   1. If the request is accepted, a recommended codec and transport address is returned by the accepting party and must be used by the initiator.
4. The channel connection is established based on the negotiation: 
   a. The initiator connects to the accepting party
   b. The accepting party connects to the initiator
   c. Both initiator and accepting party connect to the mediator
4. The stream source starts writing the stream until told to stop.
   1. The stream throughput can be regulated, if the underlying transport supports Streaming QoS. In order to get QoS feedback a bi-directional transport is required. Unidirectional transports that have no way to respond during streaming will have to accept the stream as it comes in, or break the connection.  

## Stream Data Transport

- MQTT is widely used for IoT and quite well suited for sending and receiving of data streams. Other pubsub protocols are by their nature also suitable. A channel is similar to a MQTT topic. Note that MQTT is a one to many transport while channels are primarily point to point.  

- WebRTC is a good option. It has wide platform and language support. It runs on browsers. It works with audio, video and data.

- Websockets support reading and writing. The socket path can be used as channel address.

## Use-Case: IoT Devices Connect To The Hub

IoT devices publish their data into their event stream and listen for data in their action stream. 

The device opens 3 or more channels:

1. The Hub operates a mediator 
2. The Hub has a listening endpoint for the channels: TD, event and action.
3. The Device discovers the hub mediator and identifies the endpoints it uses.
4. The Device opens 3 channels: The TD, event and action channels to the Hub.
5. The Device writes a stream of TDs on the TD channel as they are updated.
6. The Device writes a stream of events on the event channel.
7. The Device reads actions from the Hub's action channel.
4. Audio or video channels input and/or output channels, if the device supports these.

This is the most common usage and applies to all IoT devices and many services.


## Directory And History Services

The directory service has an incoming event stream for 'thing description' documents as send by IoT devices. When received, it updates its directory store with the received document.

An incoming action stream receives a request for the directory content. The service queries its store and writes the result to the response stream that is associated with the request. 

This requires that streams have identifiers, basically an address, that can be used to write into. Streams can be bidirectional or unidirectional depending on the underlying transport used.


## Authentication

The available methods of authentication depends on the underlying transport protocol. The Hub can use session tokens, signed by the Hub, that are issued during authentication. The session tokens are short lived, typically a week or two, and can be refreshed to avoid asking for a password. 
Session tokens contain the full identity of the client including the clientID and its client type 'device', 'service', or 'user'. The client type is used to determine the allowable events and actions that can be send or received on its streams.  

To obtain a session token:
1. Auto generated when connecting with a signed client certificate that contains clientID and type.
2. Login with userID and password using the underlying protocol. Eg MQTT directly supports this. One possible way to obtain the session token is to publish it on a 'session channel' that is always created when the connection is established. The channel can be used by the hub to publish connection quality of service information, pending expiry and other management information.
3. WebRTC doesn't have authentication (does it?)
