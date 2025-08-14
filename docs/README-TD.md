# HiveOT use of the TD - under development

HiveOT aims to be compliant with the WoT Thing Description specification.
The latest known draft at the time of writing is [Dec 2023](https://www.w3.org/TR/wot-thing-description11/#thing)

Interpreting this specification is done as a best effort. In case where discrepancies are reported they will be corrected when possible as long as they do not conflict with the HiveOT core paradigm of 'Things do not run servers'.

The WoT specification is closer to a framework than an application. As such it doesn't dictate how an application should use it. This document describes how the HiveOT information model and behavior maps to the WoT TD.

# HiveOT IoT Model

## What Are 'Things'?

Things are I/O Devices and Services that consumers interact with. 

Most IoT 'things' are pieces of hardware that have embedded software that manages its behavior. Virtual IoT devices build with software are also considered Things as are Services that provide a capability to retrieve information, for example a weather forecasting service.

HiveOT makes the following distinction based on the primary role of the device. These are identified by their device type as provided in the TD @type field and defined in the HiveOT vocabulary.

* A gateway is a device that provides access to other Things. A Z-Wave controller USB-stick is a gateway that uses the Z-Wave protocol to connect to Z-Wave devices. A gateway is a Thing independent of the Things it provides access to and can have its own inputs, outputs or configuration. When gateways are not WoT compatible, an agent is used as a bridge to the Hub.
* An agent is a protocol binding service that provides access to the 'Exposed Things' it manages using a WoT compatible API. HiveOT agents use connection reversal, in that they connect to the Hub instead of the other way around. 
For example, the Z-Wave agent publishes Thing information of the Z-Wave controller and the devices obtained from that controller, whose digital twin representation can be interacted with by consumers. 
* A service is a software agent that offers its capabilities as Things. For example, a history service provides a Thing that contains actions to read the history of collected data.

## Thing Description Document (TD)

The Thing Description document is a [W3C WoT standard](https://www.w3.org/TR/wot-thing-description11/#thing) to describe Things, their configuration, inputs and outputs. TDs that are published to the Hub by agents  MUST adhere to this standard and use the JSON representation format. 

The Hub digitwin (digital twin) service is a consumer of Things provided by protocol binding agents. It creates a digital twin representation of the Things it consumers. 

The hub acts as an agent of the digital twin Things it has consumed and makes the digital twin Things available to consumers. It therefore contains two TD's for each Thing, the native TD it consumed and the digital twin TD it exposes. 
The digital twin TD will have an updated Thing ID, TD Forms and authentication information.  


**TD Attributes:**

The TD documents contains a set of attributes to describe a Thing. The attributes used in HiveOT are:

| name                | description                                           |
|---------------------|-------------------------------------------------------|
| @context            | "http://www.w3.org/ns/td"                             |
| @type               | Thing device type as per vocabulary                   |
| id                  | Unique Thing ID, option in WoT but required in HiveOT |
| title               | Human readable description of the Thing (mandatory)   |
| modified            | ISO8601 date this document was last updated           |
| properties          | Map of thing attributes, state and configuration      |
| version             | Thing version as a map of {'instance':version}        |
| actions             | Map of action objects supported by the Thing          |
| events              | Map of event objects as submitted by the Thing        |
| schemaDefinitions   | data schema for use in multiple actions or events     |
| Forms               | Top level operations on Things                        |
| Links               | Definition of alternative result dataschemas          | 
| Security            | Names of security definitions                         |
| SecurityDefinitions | Security definitions for authentication with the hub  |

* As HiveOT compatible IoT devices connect to the hub, they can simply use the 'nosec' security type when creating their TD and use a NoSecurityScheme as securityDefinition. The hub will modify this section.
* Consumers, which access devices via 'Consumed Things', only need to know how connect to the Hub service. No knowledge of the IoT device protocol is needed. The TD Form sections are modified with operations to interact with a Thing's digital twin that resides on the Hub.

### @context - mandatory

The @context field defines the terminology used throughout the document which can be validated using JSON-LD. The WoT TD requires the presence of:
> https://www.w3.org/2022/wot/td/v1.1

While HiveOT provides a TD context extension with key "hiveot".

```
{
   "@context": [
   	"https://www.w3.org/2022/wot/td/v1.1",
   	{
   		"hiveot": "https://www.hiveot.net/vocab/0.1",
   	}
}
```

### @type - device type classification - optional but highly recommended

@type is a JSON-LD keyword to label the object with semantic tags, eg, add additional meaning.

HiveOT uses this field for the Thing class to help the client organize devices based on their classification.

To facilitate developers, each classification defines a minimum set of properties, events and actions to include in the TD. Additional properties, events and actions can freely be added to a TD regardless its classification.

HiveOT classification is inspired by ETSI Saref classifications, found here: https://saref.etsi.org/core/v3.1.1/. Saref's classification hierarchy seems more of an example than a complete classification and no widely adopted classification exists. For this reason hiveot defines its own classification using the "hiveot" namespace.

The basis for HiveOT's device classification is a hierarchy that can be extended with specialization terms to narrow down the classification. The full list of HiveOT device classes can be found in [api/vocab/ht-device-classes.yaml]("../api/vocab/ht-device-classes.yaml")

Top level categories in the HiveOT thing classification vocabulary are:
The vocab below is an example. Please use the vocabulary defined with the project.  
```
* hiveot:actuator - electric device for controlling an external observable feature of interest
* hiveot:appliance - class of devices for performing tasks for occupant use
* hiveot:control - devices for capturing input commands from users
* hiveot:computer - general purpose computing devices, including phones and tablets
* hiveot:media - devices for consuming or producing audio/visual media content
* hiveot:meter - metering devices for electricity, water, fuel
* hiveot:net - devices to facilitate network machine communication
* hiveot:sensor - devices designed to observe features of interest
* hiveot:service - software that processes data and offers features of interest 
```

For example:
```
* hiveot:actuator - general purpose actuator
	* hiveot:actuator:switch - electric on/off switch or relay
* hiveot:net - general purpose network management and routing device
	* hiveot:net:wifi:ap - wifi access point
* hiveot:sensor:multi - device with properties for multiple sensors such as temperature, humidity.
```
Services have their own namespace:
```
* hiveot:service - general purpose service offering capabilities
* hiveot:service:directory - directory service offering a directory of information
* hiveot:service:history - history service offering stored messages
```

For the full list of Thing classes, see [ht-thing-classes.yaml](../api/src/vocab/ht-thing-classes.yaml). Note that this list is an initial attempt for a core classification of IoT devices and services. When a more suited standard is found, it might replace this one. For this reason the vocabulary definitions are imported at runtime and mapped from their keys. See the section on vocabulary maps below.

As there are many many types of sensors and actuators, it is not the intention here to include all of them here. Instead, most specializations can be detailed through title, description and by defining device functions through events and actions. The HiveOT classification should be seen as a broad classification that makes it easy to recognize the intended purpose of the device to humans. Depth can be extended with specialization.

With the core device types standardized, the related device title and description can be provided through the vocabulary map. While still possible to include title and description in each device, the use of @type makes this unnecessary. When @type is provided the user interface uses the vocabulary classification unless a title and description is provided in the TD.

### id - the ThingID (mandatory)

The 'id' field uniquely identifies the Thing using a URI.

The WoT standard defines the ID to start with the "urn:" prefix. HiveOT uses the 'dtw:{agentID}' prefix for digital twin Things provided by agents. (still not sure if the urn: prefix is required and why. Various examples don't include it)

HiveOT bindings use the hardware device identification where possible. For example, zwave node 15 on homeID "aabbccdd" would have an 'id' value of "aabbccdd.15". When published by an agent with account ID 'zwavejs' the digitwin Thing-ID will be "dtw:zwavejs:aaabbbcc.15".

The agent-ID must be unique on the Hub. If multiple instances of an agent exist on the same hub they must have different IDs. For example, the zwave-js agent service that operates a controller has by default an id of 'zwavejs' (the service name). When using additional zwave controllers on the same hub, each instance must have a unique agent ID, eg zwavejs-1 and zwavejs-2. (Hub services use their binary name as their agent ID. Copying the binary to a different name will automatically create a new agent ID). 

Future:
When sharing digital twin things between hubs, the hub prefixes the digital twin ID with the hub ID, separated by a colon. For example, when the hub is configured to share events from node-5 of zwave-1, its events are therefore passed on to other hubs with thing ID of 'hub-1:dtw:zwave-1:node-5'. When the hub receives an action request for a thing it removes its own prefix from the thing ID and passes it to the digitwin service for further processing.

## Thing Properties

Thing Properties describe the Thing attributes, state and configuration, and are identified by their key in the TD property map.

The WoT TD describes properties with the [PropertyAffordance](https://www.w3.org/TR/wot-thing-description11/#propertyaffordance). This is a sub-class of an [interaction affordance](https://www.w3.org/TR/wot-thing-description11/#interactionaffordance) and [dataschema](https://www.w3.org/TR/wot-thing-description11/#dataschema).


The WoT specifications does not describe when to use properties. HiveOT follows these rules:

1. Static information use read-only properties.

Devices that expose information for describing the device use read-only properties to do so. For example, the manufacturer, software version, the type of device, whether it is asleep, the device model, and so on are static information.

2. State that is the result of actions is defined using read-only properties.

Stateful actions affect the internal state of the device. The state of the device is presented through read-only properties.

This avoids confusion on how to control a device. It is either through properties or actions but not both. More on this in the actions section.

3. Device configuration is defined using writable properties.

If the internal state can be updated and not the result of an action, then it is a configuration. Configurations are writable properties.

If it is an action, it is not a configuration. These are mutually exclusive.

Common configuration are alarm threshold, reporting interval, timing, device naming, for example.


### Property Name

The property name is the name in the TD property map. This name is used when publishing property value updates, events, and actions. The hub treats the names for properties, events and actions as separate names. Hence there are separate messages for publishing properties, events and actions.

Property names can be anything, although it is recommended to use a human friendly name for ease of troubleshooting. Ideally it use the vocabulary for property classification if available. If multiple instances exist then append a sequence number or other distinctive part. For example 'temperature-1'.

Property names are not used as a classification or purpose of the property. For this, take a look at the @type attribute described below.

### Property Affordance Attributes

Properties are defined with the so-called [property affordance](https://www.w3.org/TR/wot-thing-description11/#propertyaffordance). The property affordance defines a set of attributes used to describe the property.

| Attribute    | description                                                                         |
|--------------|-------------------------------------------------------------------------------------|
| @type        | property type classification (see [1])                                              | 
| type         | WoT defined data-type: string, number, integer, boolean, object, array, or null [2] |
| schema       | Name of the schema defined in the 'schemaDefinitions' section in the TD             |
| title        | Short human readable label for the property.                                        |
| description  | Longer human readable description.                                                  |
| enum         | Restricted set of values [3]                                                        |
| oneOf        | Restricted set of values with their own dataschema [3]                              |
| unit         | unit of the value                                                                   |
| readOnly     | true for Thing attributes, false for configuration                                  |
| writeOnly    | true for non-readable values like passwords.                                        |
| default      | Default value to use if no value is provided                                        |
| minimum      | type number/integer: Minimum range value for numbers                                |
| maximum      | type number/integer: Maximum range value for numbers                                |
| minLength    | type string: Minimum length of a string                                             |
| maxLength    | type string: Maximum length of a string                                             |
| *isSensor*   | the property is a sensor output (not a WoT field)                                   |
| *isActuator* | the property is an actuator (not a WoT field)                                       |
| forms        | in development [4]                                                                  |

Notes:

1. Just like the TD @type, the property @type provides a standard classification of the property. The list of core property classes are defined in the vocabulary.
2. type. WoT defines native types. HiveOT also allows the use of types defined in schemaDefinitions. This is contentious as WoT only allows schemaDefinitions in the forms 'AdditionalExpectedResponse' section. 
3. Enum values are machine identifiers. They are not translatable as this would change their value. Unfortunately the WoT TD standard does not define a method to relate an enum value to a title or description. HiveOT partly works around this by using the @type classification for presentation of enum values for standard properties.
4. With almost everything being a property, these is a need to identify whether it is a sensor, actuator, configuration or Thing attribute. 
5. Forms in development. The WoT specifies Forms to define the protocol for operations. The hub only uses top level forms. Forms in affordances are currently empty.


## Events

HiveOT uses the ground rule that changes to external state are reported through events. 

Examples of external state are environmental sensors and electrical inputs. A temperature sensor reports changes to the temperature as events. A motion sensor reports motion detection as an event.

In most cases events are not represented as properties as they do not reflect an internal state. 

Hiveot uses the @type field to classify a property but this is too fine of a granularity for this purpose and harder to apply consistently.

## Actions

### Stateless Actions

Actions can either be safe, in which case they are stateless, or unsafe when they affect internal state. 

Example of stateless actions are function calls that provide information. A weather service can provide an API that through WoT actions that returns information on the current weather.

### Stateful Actions

What is a stateful action? HiveOT defines an action as something an operator wants to control. If it is an action to the operator then it is defined as an action. While this is somewhat subjective it works as a simple ground rule for most cases. This typically relates to the primary purpose of the device. 

The most common types of actions are control of actuators, such as switches including light switches, valves, blinds and other motor controls.

State is either controlled through actions or writable properties. They are generally mutually exclusive to avoid confusion on how to control a device. As always there can be exceptions.

Action output and properties: The convention followed is that action output contains the resulting state change(s). State affected by actions have a corresponding read-only property with the same name as the action. This is not a WoT convention but is compatible for WoT consumers. 

If multiple state values are affected then both the output and the corresponding property is an object containing all affected states changes. The action output schema MUST therefor match that of the corresponding property. As this is a HiveOT convention this cannot be relied on when using non-hiveot devices. Any device presented by hiveot digital twin however follows this rule. 

If an actuator can be controlled manually, its state can change without invoking an action. This is reflected in the read-only property with the same name.

Note that this can lead to a difference between the property value and the last queryaction output. In this case the property value must be considered as actual while queryaction reflects the historical value that is the result of the last action. 



### Action Definition

The format of actions is defined in the Thing Description document
through [action affordances](https://www.w3.org/TR/wot-thing-description/#actionaffordance).

The action name is the instance name of the action. In case the device supports identical actions on multiple endpoints, like a multi-button switch, the action keys must be unique for each instance.

```
{
  "actions": {
    "{name}": ActionAffordance,
    ...
  }
}
```

Where {name} is the instance name of the action, and ActionAffordance describes the action details. The action name shares the namespace with events and properties. The main internal state affected by an action is represented with a property of the same name. If multuple states are affected, the output schema defines an object with a nested property for each affected state.

For example, the schema of an action to control an onoff switch might be defined as:

```
{
  "actions": {
    "onoff-1": {
      "title": "Control Switch 1",
      "@type": "hiveot:actuator:switch",
      "input": {
        "type": "boolean",
        "title": "turn the switch on or off"
      },
      "output": {
        "type": "boolean",
        "title": "new state of the switch"
      }
    }
  }
  "properties": {
    "onoff-1": {
       "@type": "hiveot:actuator:switch",
       "title": "Switch 1 Status",
       "type": "boolean", 
       "readOnly": true
    }
}
```
The @type attribute is defined in the vocabulary and implies an on/off switch. The title and description from the vocabulary will be used if not provided.

The action message to turn the switch on then looks like this:

```json
{
  "onoff-1": true
}
```

In HiveOT actions result in an update of the property that presents the internal state affected by the action with the same name.

Properties that reflect action output state can be read-only or writable. When writable they behave the same as actions that take only a single input parameter and affect a single state value. To avoid confusion it is convention in HiveOT to have these properties defined as read-only.  

## Thoughts, Notes, Q&A

### How to track action progress?

The consumer that initiates the action receives responses to the actions containing an ActionStatus object. Long running actions return a 'running' state and can optionally contain an intermediate output as per dataschema. When completed the last ActionStatus message contains status completed and the output as defined in the action affordance. 

If an action affects Thing state, eg properties, the action out has corresponding field names in its data schema. This is a HiveOT convention and not part of WoT.

Actuator output state changes are also reported as events with the same name as the property. This too is a HiveOT convention and not a WoT specification. 

### How to distinguish between important properties and auxiliary properties?

The WoT does not specify how to distinguish between a less and more important property. In presentation the consumer will see a long list of all possible properties. 
This is not very user-friendly so HiveOT uses the following convention to help differentiate:

1. The property @type field classifies the property usage.
2. An extra 'advanced' field indicates this is an advanced property not for common use. A user interface can hide these under normal circumstances.

Neither solution is standard though.

### How to prevent ghost actions?

Action request can take some time to be processed due to communication delays, and the ability for some devices to sleep to conserve power. The request can be delayed or aborted at any step along the way.

Actions can be time-sensitive. For example, a request to turn on a light might is likely not applicable an hour later. Queuing these requests could lead to 'ghost' actions. Ghost actions are actions that are 'lost in time' and are applied when they no longer should be.

To avoid ghost actions, action requests can carry an expiry timestamp that indicates when the request should be discarded if it cannot be handed over to the actual device in time. As a convention, HiveOT protocol bindings assume a default time window of 1 hour for actions that are queued if no expiry is provided.

Where to define the maximum lifetime or expiry of an action? Is this determined by the consumer or the Thing? A configuration property? 

### How to track the life cycle of a request?

Hiveot uses the lifecycle states of an action as defined in the WoT HTTP Basic profile:
1. pending - the request is received by the agent. The agent responds with this status immediately, even if the underlying device is not reachable.
2. running - the underlying device has received and applied the request. It has not yet completed and can still be rejected by the device. 
3. failed - something went wrong or the action was aborted.
4. completed - the agent has received confirmation that the action has been completed. This is the endpoint of the request lifecycle. The agent can reply with this status immediate or it can be sent as a lifecycle event.

Agents can immediately respond with the status pending, running, completed or failed. The status completed and failed indicate the end of the request lifecycle.

When the agent responds with pending or running, it must include a correlationID that can be used to abort a request, if supported, and to correlate them with action lifecycle events.

While intermediate lifecycle states (pending or running) are optional, a completion state - completed or failed - MUST be sent by the agent when ending the action lifecycle.

### SecuritySchema 'scheme' (1)

In HiveOT all authentication and authorization is handled by the Hub. Therefore, the security scheme section only applies to Consumer interaction with the Hub and does not apply to HiveOT Things. The security schema in the TD provided by a Thing is replaced with that of the digital twin by the Hub when publishing the digital twin TDD.

Note that it is still possible to have a protocol bindings that interacts with stand-alone Things on the network that don't adhere to the HiveOT approach. In this case this protocol binding will have to use the security schema of the Thing to connect with it. The TD from this thing will however be modified to reflect the digital twin as described above.

### Ontology and Vocabulary

HiveOT uses the Thing Description ontology from the WoT working group. WoT defines the terminology of the TD document.

Since no widely accepted vocabulary for device types and properties names have been found, HiveOT defines its own in the "hiveot" namespace. To allow for integration with other systems, the vocabulary is stored in a map which can be embedded or loaded from file. Services are coded using the 'vocab key' which is used to lookup the device or property @type, title and description before publishing the device TD. Changes to the vocabulary only needs an update of the vocabulary map. The map can also contain definitions from other 3rd providers such as Home Assistant or others. The map can be embedded or loaded from file.

The HiveOT golang library contains a handy little map container which can load an embedded vocabulary or load it from file. It contains quick lookup functions for constructing the TD.

# REST APIs

HiveOT compliant Things or agents typically do not implement their own TCP/Web servers. All interaction takes place via the HiveOT Hub using one of the transport protocol bindings. The Thing agent connects to the Hub instead of the other way around. 

The Hub implements a HTTPS/SSE protocol binding that accepts REST type requests. The REST interface is described in the Forms section of the Hub published TDDs.

