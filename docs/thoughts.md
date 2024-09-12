# Scratchpad Notes

These are some notes on thoughts and challenges that have come up.
More notes in README-TD.md

## Its All Streams Anyways

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
4. Security is not just affected by the device software but also the ecosystem it is used in. How are users managed? How are roles/permissions managed? How are the devices integrated? How is software updated? Many regular home users are just happy if they can get to see their camera on the internet and most don't have the skills needed to do this securely. 
5. There is a lack of transparency by many device manufacturers regarding the software used and known weaknesses. 

How are these issues addressed?

One part of the solution is rather simple. Do not let anything connect to the devices. If the devices don't listen for connections or can't be connected to then these connections are no longer an attack vector. Do not depend on security including user management of these devices.  

Instead, use a Hub and spokes model and let the device connect to a Hub. Hubs are nearly ubiquitous so the concept of using one is not new. What is new is that the devices only connect to this Hub instead of the other way around. This does create a new problem however, how does a device locate a Hub? This is where discovery and provisioning comes in.

A single Hub is easier to secure and update than each individual device. Authentication and user authorization can be centralized in the Hub. Again, nothing new as existing home automation Hubs already follow this model. IoT devices can remain dumb and therefore require fewer resources and can be cheaper. 

The current market trent to include full web servers on smart devices is counter this approach however. Since there is no globally accepted standard on managing these devices centrally this is understandable. Hopefully emergence of standards such as W3C WoT (Web of Things) will drive a change with device manufacturers.    

## Provisioning
2. Devices and Services need provisioning for machine based authentication. There is not an established way to provision a device so an out of band 'provisioning' protocol was made. 

## Ontology

How to describe IoT devices?

I initially established a standard called iotdomain-standard', then tried NGSI's approach but it got bulky and needs a separate server, and eventually caught up with W3C's Web of Things. So, take a look at [W3C's Thing Description](https://www.w3.org/TR/wot-thing-description11/) documentation and rejoice.

A related issue is vocabulary. Who is responsible for standardizing the terminology used in events, properties and actions?
a. The publishing device maps internal to standardized vocabulary
b. The Hub services translate the vocabulary from the publisher
c. The user interface translates.

Option a. is chosen because it constrains the mapping domain knowledge to the individual publishers. Thus a protocol binding for a particular protocol translates terminology of device events, properties and actions to standardized HiveOT vocabulary. All that consumers need to know is the standardized vocabulary.

How to define a vocabulary? Unfortunately there is no globally accepted vocabulary (yet), so one is put together using W3C or other standards. This is work in progress.

## Authorization

1. How to shape authorization? Can everyone access everything? Clearly not. HiveOT resolved this using role based authorization. A simple role system differentiate between viewers that can receive events, operators that can request actions, and administrators that can set configuration.

## Communication Protocol

1. How to communicate between devices, services, and consumers, each with their own manufactorers and implementations? What protocols to use and how to interoperate? Do consumers need to support all protocols?
  a. Initially iotdomain was message based but ran into the problem that some services require request-response. REST doesn't support callbacks and WebSockets has no standard for 2-way messaging. WoT talks a great deal about http access which is not an option as devices cannot be accessed directly. NGSI is API based.
  b. HiveOT first tried RPC using gRPC and eventually Capn'proto. Capn'proto has the more versatile IDL and turned out to be stable and very fast. gRPC is very constrained, probably good enough for Google, but too limited for defining a full API.
  c. The next iteration was messaging based using either MQTT-v5 or NATS. Both support building RPC style messages, providing best of both worlds. However, consumers need to know which one to use and http is not supported.
  d. Finally the current iteration supports multiple protocols use for agents, services and consumers. Each can use the protocols that are available to them without needing to know what the device agent protocols are. This approach is by far the most flexible of all iterations. By including the available protocols in [TDD Forms](https://www.w3.org/TR/wot-thing-description11/#form-serialization-json),  clients can choose what works best for them.

2. How to receive data real time?
    a. Message bus seems ideal here as they support subscription. So this is a no-brainer.
    b. Capn'proto supports callbacks (yeah, pretty amazing) but routing data via multiple hubs or proxies is more of a challenge.

3. Other concerns:
- high latency/non realtime of pubsub. Message round trip ideally remains under 1 msec, including audio, video and especially data.
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

- How to define API's and data types for use across languages. Protobuf has wide support but doesn't support constants and enums. capnproto is one of the few options.

Again, the W3C WoT standard has this covered. Information model serialization is defined using JSON. Forms can define the content serialization using the 'contentType' attribute.

## Message Routing

1. How to route messages? IoT data can be shared locally and possibly remotely. When semi-realtime is important, then some kind of message oriented streaming protocol is needed. What is the acceptable latency?
2. Routing messages is easier as it involves mainly addressing while RPC's are point-to-point. 
3. Some message bus protocols such as MQTT allow bridging of multiple brokers.

The above questions are resolved using the Hub as a central point of access. This does not support real-time streaming data but can be used in establishing a streaming connection between provider and consumer.

## Simplicity

The goal has always been to provide an IoT solution that is simple, secure, fast, reliable, light weight, extensible, and easy to deploy.

While the iteration using capnp RPC was fast, secure, reliable, and easy to deploy, it lacked support for javascript and required a rather impressive amount of boilerplate. 

The current 'digitwin' iteration has a simple light-weight core that meets these objectives without requiring a large server and lots of memory. Deployment is simply installation of a few binaries without the need to run docker containers. A mid-sized local hub runs fine on a raspberry pi 2 with 1GB of memory. Of course it is always possible to deploy using containers. 


## Lack of Audio And Video

Support for audio and video is desirable but is not supported by IoT protocols.
Streaming media is best done using their own protocols due to their latency and cpu requirements. The hub is however well suited for setting up media streams. A protocol binding for this purpose is planned.


## WoT TD

While the WoT TD covers a lot of use-cases, there are few workarounds needed:

1. Enum values have no label/title. 
   The workaround is to use oneOf instead.
2. No best practices with examples for various use-cases. 
   This is being addressed and a [developer resources website](https://www.w3.org/WoT/developers/) has been added. 
3. No IoT vocabulary, everyone has to roll their own.
   This remains an issue. The workaround is to establish a hiveot vocabulary using the "ht:" context prefix. Some better alternatives might be available but this need further investigation.  
4. Interaction affordances are difficult to implement in strongly typed languages like golang. The inheritance model cannot be implemented. For example, a PropertyAffordance inherits from dataschema but each data type has its own dataschema. 
    The workaround is to define a 'flat' dataschema that contains all properties of all dataschemas. Ugly but it works with a minor naming collision.
5. Multiple types for same property (single item vs list)
    The current workaround is to use 'any' type for those properties that can be either a string or list and use inspection in runtime to determine which one to use. Yes, yuckie.  
6. Forms/protocols unclear (2023). This is mostly due to my ignorance and will be resolved soon. WoT folks have been very helpful in providing guidance. (Thank you!)
    Things do get a bit wordy though as each action/event/property will have a set of forms, one for each supported protocol.  
* ThingID uniqueness; using agentID in address 
    This is resolved by letting agents use anything as thingID as long as only valid characters are used, and have the digital twin runtime prefix the thing-ID it with the agent ID.



## Encoding data

1. agent publishes a TD
A TD is send and received as JSON. But this text is still to be marshalled and transferred over the wire.
   Treat this as sending text
if encoding is application/json then use json.Marshal(tdJSON) - (yes double marshal)
if encoding is application/text then send as-is. (it is already text)
 

1. agent publishes native type: text, number, boolean
if encoding is application/json, then marshal as json
if encoding is application/text, then stringify as text, 
examples:
   * application/text:  "text" => "text"
   * application/text:  8 => "8"
   * application/text:  true => "true"

1. agent publishes complex type: object
if encoding is application/json then marshal object to json
if encoding is application/text then write as base64 encoded binary blob. (how?)
examples:
   * application/json:  struct { a int, b string} => { "a": 5, "b", "hello" } 
   

## Decoding data

1. consumer or service receives a TD message
The TD was transferred as text (itself encoded as a json document).
If encoding is application/json, the unmarshal to string
If encoding is application/text, then return as-is. 



## Tracking delivery progress
Action requests can take a while to reach its destination. The delivery to the inbox is immediately known but what the agent does with it isn't.

> ACTION flow: consumer -> inbox ?-> agent ?-> device

* Agent connectivity might be intermittent. Some agents might sleep periodically to conserve power.  
* Similarly to agents, devices might be asleep to conserve power. Only when it wakes up it will respond to the action.
* High latency networks favour asynchronous handling of responses to avoid blocking the client.

This means that the reply isn't always readily available and can be delivered later. To support these use-cases, reply data is sent asynchronously.