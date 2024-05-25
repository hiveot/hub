# HiveOT use of the TD - under development

HiveOT intends to be compliant with the WoT Thing Description specification.
The latest known draft at the time of writing is [Mar 2022](https://www.w3.org/TR/wot-thing-description11/#thing)

Interpreting this specification is done as a best effort. In case where discrepancies are reported they will be corrected when possible as long as they do not conflict with the HiveOT core paradigm of 'Things do not run servers'.

The WoT specification is closer to a framework than an application. As such it doesn't dictate how an application should use it. This document describes how the HiveOT information model and behavior maps to the WoT TD.

# HiveOT IoT Model

## What Are 'Things'?

Things are I/O Devices and services that provide information. 

Most IoT 'things' are pieces of hardware that have embedded software that manages its behavior. IoT devices build with software are also considered Things as are services that provide a capability to retrieve information, for example a weather forecasting service.

HiveOT makes the following distinction based on the primary role of the device. These are identified by their device type as provided in the TD @type field and defined in the HiveOT vocabulary.

* A gateway is a device that provides access to other Things. A Z-Wave controller USB-stick is a gateway that uses the Z-Wave protocol to connect to Z-Wave devices. A gateway is a Thing independent of the Things it provides access to and can have its own inputs, outputs or configuration.
* An agent is a Thing that publishes Thing information to the HiveOT Hub. A publisher has authorization to publish and subscribe to the things it is the publisher of. For the Z-Wave agent publishes Thing information of the Z-Wave controller and the devices obtained from that controller.
* An I/O device is a Thing whose primary role is to provide access to inputs and outputs and has its own attributes and configuration.
* A Hub bridge is a software device that connects two Hubs and shares configured Thing information between them.
* A service is a software Thing that offers a capability in the IoT ecosystem. For example, a history service provides a history of Thing values.

## Thing Description Document (TD)

The Thing Description document is a [W3C WoT standard](https://www.w3.org/TR/wot-thing-description11/#thing) to describe Things, their configuration, inputs and outputs. TDs that are published on the Hub MUST adhere to this standard and use the JSON representation format.

**TD Attributes:**

The TD documents contains a set of attributes to describe a Thing. The attributes used in HiveOT are:

| name       | mandatory | description                                           |
|------------|-----------|-------------------------------------------------------|
| @context   | mandatory | "http://www.w3.org/ns/td"                             |
| @type      | optional  | Thing device type as per vocabulary                   |
| id         | required  | Unique Thing ID, option in WoT but required in HiveOT |
| title      | mandatory | Human readable description of the Thing               |
| modified   | optional  | ISO8601 date this document was last updated           |
| properties | optional  | Map of thing attributes, state and configuration      |
| version    | optional  | Thing version as a map of {'instance':version}        |
| actions    | optional  | Map of action objects supported by the Thing          |
| events     | optional  | Map of event objects as submitted by the Thing        |

note: Consumers do not connect directly to the IoT device. Communication protocols, authentication & authorization is handled by the Hub services. As a result, forms and security definitions in the TD do not appy to the IoT device and are not used in the TD.

* HiveOT compatible IoT devices can simply use the 'nosec' security type when creating their TD and use a NoSecurityScheme as securityDefinition.
* Consumers, which access devices via 'Consumed Things' (a remote instance of the thing with properties and values), only need to know how connect to the Hub service. No knowledge of the IoT device protocol is needed.

### @context - mandatory

The @context field defines the terminology used throughout the document which can be validated using JSON-LD. The WoT TD requires the presence of:
> https://www.w3.org/2022/wot/td/v1.1

While HiveOT provides a TD context extension with key "ht".

```
{
   "@context": [
   	"https://www.w3.org/2022/wot/td/v1.1",
   	{
   		"ht": "https://www.hiveot.net/vocab/0.1",
   	}
}
```

### @type - device type classification - optional but highly recommended

@type is a JSON-LD keyword to label the object with semantic tags, eg, add additional meaning.

HiveOT uses this field for the Thing class to help the client organize devices based on their classification.

To facilitate developers, each classification defines a minimum set of properties, events and actions to include in the TD. Additional properties, events and actions can freely be added to a TD regardless its classification.

HiveOT classification is inspired by ETSI Saref classifications, found here: https://saref.etsi.org/core/v3.1.1/. Saref's classification hierarchy seems more of an example than a complete classification and no widely adopted classification exists. For this reason hiveot defines its own classification using the "ht" namespace.

The basis for HiveOT's device classification is a hierarchy that can be extended with specialization terms to narrow down the classification. The full list of HiveOT device classes can be found in [api/vocab/ht-device-classes.yaml]("../api/vocab/ht-device-classes.yaml")

Top level categories in the HiveOT thing classification vocabulary are:
```
* ht:actuator - electric device for controlling an external observable feature of interest
* ht:appliance - class of devices for performing tasks for occupant use
* ht:control - devices for capturing input commands from users
* ht:computer - general purpose computing devices, including phones and tablets
* ht:media - devices for consuming or producing audio/visual media content
* ht:meter - metering devices for electricity, water, fuel
* ht:net - devices to facilitate network machine communication
* ht:sensor - devices designed to observe features of interest
* ht:service - software that processes data and offers features of interest 
```

For example:
```
* ht:actuator - general purpose actuator
	* ht:actuator:switch - electric on/off switch or relay
* ht:net - general purpose network management and routing device
	* ht:net:wifi:ap - wifi access point
* ht:sensor:multi - device with properties for multiple sensors such as temperature, humidity.
```
Services have their own namespace:
```
* ht:service - general purpose service offering capabilities
	* ht:service:directory - directory service offering a directory of information
	* ht:service:history - history service offering stored messages
```

For the full list of Thing classes, see [ht-thing-classes.yaml](../api/src/vocab/ht-thing-classes.yaml). Note that this list is an initial attempt for a core classification of IoT devices and services. When a more suited standard is found, it might replace this one. For this reason the vocabulary definitions are imported at runtime and mapped from their keys. See the section on vocabulary maps below.

As there are many many types of sensors and actuators, it is not the intention here to include all of them here. Instead, most specializations can be detailed through title, description and by defining device functions through events and actions. The HiveOT classification should be seen as a broad classification that makes it easy to recognize the intended purpose of the device to humans. Depth can be extended with specialization.

With the core device types standardized, the related device title and description can be provided through the vocabulary map. While still possible to include title and description in each device, the use of @type makes this unnecessary. When @type is provided the user interface uses the vocabulary classification unless a title and description is provided in the TD.

### id - the ThingID (mandatory)

The 'id' field uniquely identifies the Thing using a URI.

The WoT standard requires the ID to start with the "urn:" prefix. HiveOT doesn't care as this rule doesn't seems to be providing a capability used by HiveOT. However, for the sake of consistency it applies this rule as well.

HiveOT uses the hardware device identification where possible. For example, zwave node 15 on homeID "aabbccdd" would have an 'id' value of "urn:aabbccdd-15".

Since thing IDs are defined by the device or service they are not guaranteed to be globally unique.

In case of protocol binding services, also called agents, Things are addressed indirectly via the agent that published the Thing information. For example, to address a zwave node, both the zwave agent ID and the node Thing ID are needed. As a benefit, separating these parts makes it simpler to filter a group of Things based on their agent.

The agent is also a Thing and has its own thingID. If multiple instances of an agent exist on the same hub they must have different IDs. For example, the zwave agent service that operatess a controller has by default an id of 'zwave' (the service name). When using two zwave controllers on the same hub, each instance must have a unique agent ID, eg zwave-1 and zwave-2. The easiest way to accomplish this is to rename the agent binary and copy it as the default ID is the binary name. Each binary will get its own thing with its own configuration and authentication. Note that this is required whether the service lives on the same system or two systems.

When sharing things between hubs, the hub prefixes the agent ID with its own hub ID, separated by a colon. So 'hub-1:zwave-1' is the agent ID for all zwave-1 Things from the hub with ID 'hub-1'. When the hub is configured to share events from node-5 of zwave-1, its events are therefore passed on to other hubs with agent ID of 'hub-1:zwave-1' and thing ID 'node-5'. When the hub receives an action request for a thing it removes its own prefix from the publisher ID and passes it to the agent that matches the remainder of the agent ID, eg 'zwave-1'.

## Thing Properties

Thing Properties describe the Thing attributes, state and configuration, and are identified by their key in the TD property map.

The WoT TD describes properties with the [PropertyAffordance](https://www.w3.org/TR/wot-thing-description11/#propertyaffordance). This is a sub-class of an [interaction affordance](https://www.w3.org/TR/wot-thing-description11/#interactionaffordance) and [dataschema](https://www.w3.org/TR/wot-thing-description11/#dataschema).

HiveOT uses the following 'Property' attributes to describe properties:

### Property ID

The property ID is the key in the TD property map. This ID is used when publishing property value updates, events, and actions. The same property key can be used in the event and action affordance map when they are related to the same property.

Property ID's can be anything, although it is recommended to use the property classification. If multiple instances exist then append a sequence number or other distinctive part. For example 'temperature-1'.

Property IDs are not used as a classification or purpose of the property. For this, take a look at property @type attribute described below.

### Property Attributes

Properties are defined with the so-called [property affordance](https://www.w3.org/TR/wot-thing-description11/#propertyaffordance). The property affordance defines a set of attributes used to describe the property.

| Attribute   | WoT      | description                                                                                       |
|-------------|----------|---------------------------------------------------------------------------------------------------|
| @type       | optional | property type classification (see [1])                                                            | 
| type        | optional | WoT defined data-type: string, number, integer, boolean, object, array, or null                   |
| title       | optional | Short human readable label for the property. Not needed if property @type classification is used. |
| description | optional | Longer human readable description. Not needed if property @type classification is used.           |
| enum        | optional | Restricted set of values [2]                                                                      |
| unit        | optional | unit of the value                                                                                 |
| readOnly    | optional | true for Thing attributes, false for configuration                                                |
| writeOnly   | optional | true for non-readable values like passwords.                                                      |
| default     | optional | Default value to use if no value is provided                                                      |
| minimum     | optional | type number/integer: Minimum range value for numbers                                              |
| maximum     | optional | type number/integer: Maximum range value for numbers                                              |
| minLength   | optional | type string: Minimum length of a string                                                           |
| maxLength   | optional | type string: Maximum length of a string                                                           |                               
Notes:

1. Just like the TD @type, the property @type provides a standard classification of the property. The list of core property classes are defined in the vocabulary.
2. Enum values are machine identifiers. They are not translatable as this would change their value. Unfortunately the WoT TD standard does not define a method to relate an enum value to a title or description. HiveOT partly works around this by using the @type classification for presentation of enum values for standard properties.
3. Forms are not used. The WoT specifies Forms to define the protocol for operations. In HiveOT all operations operate via a message bus with a simple address scheme. There is therefore no use for Forms. In addition, requiring a Forms section in every single property description causes unnecessary bloat that needs to be generated, parsed and stored by exposed and consumed things.
4. In HiveOT the namespace for properties, events and actions is shared. A property ID MUST refer to the same device attribute in the properties, events and actions map.
5. The use of readOnly and writeOnly attributes can be somewhat ambiguous. In HiveOT, if 'writeOnly' is true then the value can be set but not read and the value of readOnly is ignored. For example a password can be written but not read. When both writeOnly and readOnly are false then the property is writable.

## Events

Where properties contain the last known state value of a Thing, events are used to notify of changes to the state.

Events are primarily defined for values that related to the purpose of the Thing. For example, a multisensor device has events for each of the sensor values, but not for the individual attributes or configuration properties. Additional events can be defined for values that are critical to operations, for example a low battery event.

> Note: In many cases a corresponding property exists with the same ID that contains the most recent value.
> Note: a separate 'properties' event is sent periodically containing a map of all properties that have changed. The delay between a change and sending an update should not exceed 1 minute.

The TD events affordance section defines the attributes used to describe events. This is similar to how properties are defined.

TD Events map:
```
{
  "events": {
    "{eventID}": {
      ...InteractionAffordance,
      "data": {
        dataSchema
      },
      "dataResponse: {EventResponseData}"
    }
  }
}
```

Where:

* {eventID}: The instance ID of the event. Event IDs follow the same naming convention as the property IDs. Often, an event has a corresponding property that contains the most recent value.
* @type: event/property type classification. While not mandatory it is highly recommended. See the property classification for details.
* data: The event payload follows the [dataSchema](https://www.w3.org/TR/wot-thing-description11/#dataschema) format, similar to properties.

The [TD EventAffordance](https://www.w3.org/TR/wot-thing-description11/#eventaffordance) also describes optional event response, subscription and cancellation attributes. These are not used in HiveOT as subscription is handled by the Hub.

### The "properties" Event

In HiveOT, changes to property values are sent using the "properties" event. Rather than sending a separate event for each property, HiveOT bundles changes to properties into a 'properties' event. This event contains a map of property ID-value pairs.

The 'properties' event can be delayed up to a minute to allow time for collecting multiple changes to be included. This is an optimization that can be configurable.

As the 'properties' event is part of the HiveOT standard, it does not have to be included in the 'events' section of the TDs (but is recommended).  (should this be part of a HiveOT @context? tbd)

TD 'properties' event description:
```json
{
  "events": {
    "properties": {
      "data": {
        "title": "Map of property name and new value pairs",
        "type": "object"
      }
    }
  }
}
```

For example, when the 'name' property has changed value, the corresponding properties event content looks like:

```
{
  "name": "new name",
}
```

## Actions

Actions are used to control inputs and change the value of configuration properties.

The format of actions is defined in the Thing Description document
through [action affordances](https://www.w3.org/TR/wot-thing-description/#actionaffordance).

Similar to events, the actionID is the instance name of the action. By default this matches the action's @type classification from the vocabulary. In case the device supports multiple actions of the same type, like a multi-button switch, the action ID contains the instance ID while the '@type' attribute contains the action type classification.

```
{
  "actions": {
    "{actionID}": ActionAffordance,
    ...
  }
}
```

Where {actionID} is the instance ID of the action, and ActionAffordance describes the action details. The action ID shares the namespace with events and properties. The result of an action can be notified using an event with the same name and shown with a property of the same name:

```
{
  ...interactionAffordance, // including @type classification
  "input": {},
  "output": {},
  "safe": true
  |
  false,
  "idempotent": true
  |
  false
}
```

For example, the schema of an action to control an onoff switch might be defined as:

```
{
  "actions": {
    "onoff-1": {
      "title": "Control the on or off status of a switch",
      "@type": "ht:actuator:switch",
      "input": {
        "type": "boolean",
        "title": "turn the switch on or off"
      }
    }
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

### The 'properties' Action

Similar to a properties Event, HiveOT standardizes a "properties" action. To change a configuration value, a properties action must be submitted containing a map of property ID-value pairs to update. When defining the 'hiveot' @context in the TD, no additional action affordance is needed to write properties. (TODO describe how to do this)

For example, when the Thing configuration property called "alarmThreshold" changes, the 'properties' action payload contains:

```json
{
  "alarmThreshold": 25
}
```

## Notes and Q&A

### How to handle events, actions and properties with the same name?

Sometimes the same property ID applies to an event and action. How should the agent create the TD to handle this?

For example, the onoff switch for controlling a light has a property with the current state, an event when the state changes and an action to request setting the new state. Perhaps even a writable property for controlling the state through configuration.

This can work seamlessly without conflict. A property, event and action affordance is defined all with the same ID, each in their respective map in the TD document. When an action request is received, the action affordance is used for constructing and validating the request. After the request is accepted and the device changes state, the corresponding property is updated and an event with the ID is published. If both a property and an event exist with the same name then two events are sent, one with the event ID and the second being the 'properties' event. Not that the properties event can be delayed to gather multiple successive changes.

The main criteria is that the ID used in the property, event, and action map refers to the same internal state of the device. If they differ then a different ID should be used.

For example, a media player has a property of @type 'ht:prop:media:muted'. This can be used directly as a property ID. When the device internal muted state changes to true, an event with the id 'ht:prop:media:muted' is published. However, the action to mute the player would use a different ID: "ht:action:media:mute", since the action is only a request and does not represent the state of the device. If accepted the event will confirm the request.

### How to track the progress of an action and prevent ghost actions?

Handling action requests is complicated due to the multiple intermediaries that are involved, the delayed handling due to communication delays, and the ability for some devices to sleep to conserve power. The request can be delayed or aborted at any step along the way.

TODO: The following is a proposal on handling action lifecycles.

Many action requests are time sensitive. For example, a request to turn on a light might is likely not applicable an hour later. This could lead to 'ghost' actions. Ghost actions are actions that are 'lost in time' and are applied when they no longer should be.

To avoid ghost actions, each action request carries an expiry timestamp that indicates when the request should be discarded if it cannot be handed over to the actual device in time.

todo: where to define the expiry? transport envelope?

The lifecycle states of an action are:
1. requested - the request is sent but not yet acknowledged by the agent. This state is only known on the sending client.
2. pending - the request is accepted by the agent. The agent responds with this status immediately, even if the underlying device is not reachable.
3. accepted - the underlying device has accepted the request but not yet confirmed that it was applied. The agent can reply with this status immediately or it can be sent as a lifecycle event.
4. success - the agent has received confirmation that the action has been applied. This is the endpoint of the request lifecycle. The agent can reply with this status immediate or it can be sent as a lifecycle event.

When things don't go to plan:
5. aborted - the request is aborted by the user or agent, or has expired.

Agents can immediately respond with the status pending, accepted, success or aborted. The status success and aborted indicate the end of the request lifecycle.

When the agent responds with pending or accepted, it must include a request ID that can be used to abort a request, if supported, and to correlate them with action lifecycle events.

While intermediate lifecycle events (pending or accepted) are optional, a completion event - success or aborted - MUST be sent by the agent when ending the action request lifecycle.

Action lifecycle events are events with the eventID lifecycle (see vocabulary) and a payload containing the action ID and the new status.

```json
{
  "actionID": "...",
  "status": "accepted|success|aborted",
  "data": {}
}
```

Where 'data' is optional response data to the action as defined in the TD action affordance, or the reason when the request is aborted.

### Working With Enums

WoT's DataSchema (which is the base for properties, events and actions) includes an 'enum' field that is defined as an array of any type. It is intended to contain a restricted set of values.

Unfortunately, while WoT's DataSchema does define an 'enum' value array for a property/event/action, it does not provide a way to define the title and description for these values. So how to present these enum values?

This is has been discussed in the WoT group [here](https://github.com/w3c/wot-thing-description/issues/997#issuecomment-1865902885) where the proposed solution is the use 'oneOf', or to add an 'enumMap' attribute.

HiveOT chooses to use oneOf with an array of DataSchema to support enums with DataSchema annotations. The 'const' field defines the value while 'title' provides a human description. The use of '@type' is possible in case of a value from another vocabulary.

### Links

The TD specification describes a [link](https://www.w3.org/TR/wot-thing-description11/#link) as "A link can be
viewed as a statement of the form "link context has a relation type resource at link target", where the optional target attributes may further describe the resource"

HiveOT ignores links.

### Forms

The WoT specification for a [Form](https://www.w3.org/TR/wot-thing-description11/#form) says: "A form
can be viewed as a statement of "To perform an operation type operation on form context, make a
request method request to submission target" where the optional form fields may further describe the required request."

HiveOT ignores forms, although support might be added in the future for allowing control via different protocol bindings. Agents can define form items but as they are not allowed to be servers they cannot provide direct endpoints.

### SecuritySchema 'scheme' (1)

In HiveOT all authentication and authorization is handled by the Hub. Therefore, the security scheme. This section only applies to Hub interaction and does not apply to HiveOT Things. The Hub service used to interact with Things will publish a TD that includes a SecuritySchema needed to interaction with the Hub.

### Ontology and Vocabulary

HiveOT uses the Thing Description ontology from the WoT working group. WoT defines the terminology of the TD document.

Since no widely accepted vocabulary for device types and properties names have been found, HiveOT defines its own in the "ht" namespace. To allow for integration with other systems, the vocabulary is stored in a map which can be embedded or loaded from file. Services are coded using the 'vocab key' which is used to lookup the device or property @type, title and description before publishing the device TD. Changes to the vocabulary only needs an update of the vocabulary map. The map can also contain definitions from other 3rd providers such as Home Assistant or others. The map can be embedded or loaded from file.

The HiveOT golang library contains a handy little map container which can load an embedded vocabulary or load it from file. It contains quick lookup functions for constructing the TD.

# REST APIs

HiveOT compliant Things or agents do not implement TCP/Web servers. All interaction takes place via the HiveOT Hub using the message bus. Therefore, this section only applies to specialized Hub services that provide a web API.

Hub services that implement a REST API follows the approach as described in Mozilla's Web Thing REST API](https://iot.mozilla.org/wot/#web-thing-rest-api).

```http
GET https://address:port/things/{thingID}[/...]
```

The WoT examples often assume or suggest that Things are directly accessed, which is not allowed in HiveOT. Therefore, the implementation of this API in HiveOT MUST adhere to the following rules:

1. The Thing address is that of the hub it is connected to.
2. Both agentID and thing ID must be included in the API. A thing is not addressable without its agent. 
