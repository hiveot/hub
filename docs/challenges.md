# Its All Streams Anyways

Considerations and learnings from the evolution of hiveot...

The HiveOT Hub has gone through several iterations, starting with the pure mqtt/mosquitto based network of devices and services (iotdomain), NGSI based things, and eventually landing on the W3C Web Of Things. Communication methods included MQTT pubsub, REST API for services, gRPC and Capn'proto RPC based services. 

Each of these approaches ran into various challenges. Below a short summary of these challenges and how they have been addressed.

## Security

Security is perhaps the biggest challenge. How to get a grip on security? 

Many IoT devices are notoriously insecure and often lack the neccesary security updates. The MiRA botnet that uses internet exposed cameras is just the tip of the iceberg. The potential damage due to abuse of internet exposed devices is enormous. 

What is the problem?
1. Devices either do not get security updates or they are not applied. Exposing devices with security holes to the internet can quickly get them to be Pwnd by attackers. 
2. The cost of building, testing and updating secure software is high and for cheap devices it can be prohibitive.  
3. Security is hard. The knowledge needed to build a fully secure IoT device is extensive and not many people know it all.
4. Security is not just affected by the device software but also the ecosystem it is used in. How are users managed? How are roles/permissions managed? How are the devices integrated? How is software updated? Many regular home users are just happy if they can get to see their camera on the internet and many don't have the skills needed to do this securely. Arguably they shouldn't need to but the reality is  different.
5. The information produced is not verifyable. 

How are these issues addressed?

One part of the solution is rather simple. Do not let anything connect to the devices. If the devices don't listen for connections or can't be connected to then these connections are no longer an attack vector. Do not depend on security including user management of these devices.  

Instead, use a Hub and spokes model and let the device connect to a Hub. Hubs are nearly ubiquitous so the concept of using one is not new. What is new is that the devices only connect to this Hub instead of the other way around. This does create a new problem however, how does a device locate a Hub? This is where provisioning comes in.

A single Hub is easier to secure and update than each individual device. Authentication and user authorization can be centralized in the Hub. Again, nothing new as existing home automation Hubs already follow this model. IoT devices can remain dumb and therefore require fewer resources and can be cheaper.

## Provisioning
2. Devices and Services need provisioning for machine authentication. There is not an established way to provision a device so an out of band 'provisioning' protocol was made. 

## Ontology

How to define IoT devices?

Initially established a standard called iotdomain-standard', then tried NGSI's approach but it got bulky and needs a separate server, and eventually caught up with W3C's Web of Things.

## Authorization

1. How to shape authorization? Can everyone access everything? Clearly not. HiveOT resolved this using groups where members can access Things in that group. A simple role system differentiate between viewers that can receive events, controllers that can request actions, and administrators that can set configuration.

## Communication Protocol

1. How to communicate between devices and Hub services, between users and Hub, and between multiple Hubs? Use messaging, REST, or RPC for communication? 
   2. Initially iotdomain was message based but ran into the problem that some services require request-response. REST doesn't support callbacks and WebSockets has no standard for 2-way messaging. WoT talks a great deal about http access which is not an option as devices cannot be accessed directly. NGSI is API based.

   2. HiveOT tried RPC using gRPC and eventually Capn'proto. Capn'proto has the more versatile IDL and turned out to be stable and very fast. gRPC is very constrained, probably good enough for Google, but limited for defining a full API.
1. How to receive data real time?
    2. Message bus seems ideal here as they support subscription.
    2. Capn'proto supports callbacks (yeah, pretty amazing) but routing data via multiple hubs is more of a challenge.
- high latency/non realtime of pubsub. Message round trip should remain under 1 msec, including audio, video and especially data.
- not always confirmation of delivery (messaging vs rpc)
- Lack of support for audio and video.
- session based vs send and forget (rpc vs messaging)
- monoliths
- multiple language support


## Data Serialization

1. How to serialize data? Boilerplate code for RPC wrappers can get tedious, especially capnp requires a lot of code. Every single RPC api needs boilerplate for parameter marshalling and unmarshalling.
   2. Message based approach has less boilerplate as only the data definitions need to be defined, not the API parameter/responses that are used to pass those messages. 
   3. There are also more ways to define messages vs RPC APIs. The serialization of messages can be configurable as the transport doesn't care. 
   4. gRPC and capn'proto tie the serialization to the RPC and its hard to separate.  

- Limited performance of serialization. json is slow while capnproto is fast but requires a lot of coding. Protobuf is somewhere in the middle but still requires a significant amount of boilerplate.


- How to define API's and data types for use across languages. Protobuf has wide support but is very limited. It doesn't even support constants. and capnproto are one of the few options

## Message Routing

1. How to route messages? IoT data can be shared locally and possibly remotely. When semi-realtime is important, then some kind of message oriented streaming protocol is needed. What is the acceptable latency?
   2. Routing messages is easier as it involves mainly addressing while RPC's are point-to-point. 
   3. Some message bus protocols such as MQTT allow bridging of multiple brokers. 

## Simplicity

The goal has always been to provide an IoT solution that is simple, secure, fast, reliable, light weight, extensible, and easy to deploy.

While the last iteration using capnp RPC was fast, secure, reliable, and easy to deploy, it lacked support for javascript and required a rather impressive amount of boilerplate. 


## Lack of Audio And Video

Support for audio and video is desirable but is not supported by IoT protocols.