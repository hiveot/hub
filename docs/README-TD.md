# HiveOT use of the TD - under development

HiveOT intends to be compliant with the WoT Thing Description specification.
The latest known draft at the time of writing is [Mar 2022](https://www.w3.org/TR/wot-thing-description11/#thing)

Interpreting this specification is done as a best effort. In case where discrepancies are reported they will be corrected when possible as long as they do not conflict with the HiveOT core paradigm of 'Things do not run servers'.

The WoT specification is closer to a framework than an application. As such it doesn't dictate how an application should use it. This document describes how the HiveOT information model and behavior maps to the WoT TD.

# HiveOT IoT Model

## What Are 'Things'?

I/O Devices, Gateways, Publishers, Services are IoT 'Things'.

Most IoT 'things' are pieces of hardware that have embedded software that manages its behavior. Virtual IoT devices are build with software only but are otherwise considered identical to hardware devices. Services can also be provided as Things.

IoT devices often fulfill multiple roles: a part provides network access, a part provides access to inputs and outputs, a part reports its state, and a part that manages its configuration.

HiveOT makes the following distinction based on the primary role of the device. These are identified by their device type as provided in the TD @type field and defined in the HiveOT vocabulary.

* A gateway is a Thing that provides access to other Things. A Z-Wave controller USB-stick is a gateway that uses the Z-Wave protocol to connect to I/O devices. A gateway is independent of the Things it provides access to and can have its own inputs or outputs. 
* A publisher is a Thing that publishes Thing information to the HiveOT Hub. A publisher has authorization to publish and subscribe to the things it is the publisher of. For example, a gateway is the publisher of the Things obtained through the gateway. 
* An I/O device or service is a Thing whose primary role is to provide access to inputs and outputs and has its own attributes and configuration. 
* A Hub bridge is a device that connects two Hubs and shares Thing information between them.
* A service is software that offers a capability in the IoT ecosystem. For example, a directory service stores TD documents. Services are also Things.  


## Thing Description Document (TD)

The Thing Description document is a [W3C WoT standard](https://www.w3.org/TR/wot-thing-description11/#thing) to describe Things, their configuration, inputs and outputs. TDs that are published on the Hub MUST adhere to this standard and use the JSON representation format.


## TD Attributes

The TD documents contains a set of attributes to describe Things. The attributes used in HiveOT are:

| name               | mandatory | description                                           |
|--------------------|-----------|-------------------------------------------------------|
| @context           | mandatory | "http://www.w3.org/ns/td"                             |
| @type              | optional  | Thing device type as per vocabulary                   |
| id                 | required  | Unique Thing ID, option in WoT but required in HiveOT |
| title              | mandatory | Human readable description of the Thing               |
| modified           | optional  | ISO8601 date this document was last updated           |
| properties         | optional  | Map of thing attributes, state and configuration      |
| version            | optional  | Thing version as a map of {'instance':version}        |
| actions            | optional  | Map of action objects supported by the Thing          |
| events             | optional  | Map of event objects as submitted by the Thing        |

note: Consumers do not connect directly to the IoT device. Communication protocols, authentication & authorization is handled by the Hub services. As a result, forms and security definitions in the TD do not appy to the IoT device and are not used in the TD.

* HiveOT compatible IoT devices can simply use the 'nosec' security type when creating their TD and use a NoSecurityScheme as securityDefinition.
* Consumers, which access devices via 'Consumed Things', only need to know how connect to the Hub service. No knowledge of the IoT device protocol is needed.


## Thing Properties

Thing Properties describe the Thing attributes, state and configuration.

The WoT TD describes properties with the [PropertyAffordance](https://www.w3.org/TR/wot-thing-description11/#propertyaffordance). This is a sub-class of an [interaction affordance](https://www.w3.org/TR/wot-thing-description11/#interactionaffordance) and [dataschema](https://www.w3.org/TR/wot-thing-description11/#dataschema).

HiveOT uses the '@type' property attribute to distinguish between attr, state and config, where attr contain static attributes such as product make and model; state reflects the operating state such as battery level, and config are writable properties like 'name'. 

HiveOT uses the following attributes to describe properties.

| Attribute   | WoT       | description                                                               |
|-------------|-----------|---------------------------------------------------------------------------|
| name        | optional  | Name used to identify the property in the TD Properties object. (1)       |
| @type       | optional  | property type 'attr', 'config', 'state'                                   | 
| type        | optional  | data type: string, number, integer, boolean, object, array, or null       |
| title       | optional  | Human description of the property.                                        |
| description | optional  | In case a more elaborate description is needed for humans                 |
| enum        | optional  | Restricted set of values                                                  |
| unit        | optional  | unit of the value                                                         |
| readOnly    | optional  | true for properties that are attributes, false for writable configuration |
| writeOnly   | optional  | Used to set non-readable values like passwords.                           |
| default     | optional  | Default value to use if no value is provided                              |
| minimum     | optional  | type number/integer: Minimum range value for numbers                      |
| maximum     | optional  | type number/integer: Maximum range value for numbers                      |
| minLength   | optional  | type string: Minimum length of a string                                   |
| maxLength   | optional  | type string: Maximum length of a string                                   |                               
Notes:

1. Common property names are standardized as part of the vocabulary so consumers can understand their purpose.
2. Forms are not used. The WoT specifies Forms to define the protocol for operations. In HiveOT all operations operate via a message bus with a simple address scheme. There is therefore no use for Forms. In addition, requiring a Forms section in every single property description causes unnecessary bloat that needs to be generated, parsed and stored by exposed and consumed things.
3. In HiveOT the namespace for properties, events and actions is shared to avoid ambiguity. A change in property value can lead to an event with the property name. Writing a property value is done with an action of the same name. (the WoT group position on this is unknown. Is this intended?)
4. The use of readOnly and writeOnly attributes is somewhat unclear. In HiveOT, 'writeOnly' means that the value can be set but not read. For example a password. When readOnly equals false this implies writable.


## Events

Changes to the outputs of a Thing are published using Events. An output can be a sensor value, actuator state change, or notification. The TD describes an event for each output. 

The event name is the output type, such as 'temperature' and 'switch'. These names are standardized in the vocabulary. In case of multiple outputs of the same type the name is appended with the instance ID. For example, 'temperature-1'. In this case the @type attribute of the event also contains the actual output type. The TD events affordance section defines the attributes used to describe events. 

TD Events section: 
```
{
  "events": {
    "{eventName}": {
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

* {eventName}: The instance name of the event. Event names share the same namespace as property names. By default this is the standardized event type name from the HiveOT vocabulary.  In case of multiple instances of the same event type, like multiple switches, the instance nr is appended. For example "temperature-1" or "scene-3". 
* @type: event type name, like 'temperature' or 'scene'. Can be omitted if the event name is also the type. 
* data: Defines the data schema of event messages. The content follows
  the [dataSchema](https://www.w3.org/TR/wot-thing-description11/#dataschema) format, similar to
  properties.

The [TD EventAffordance](https://www.w3.org/TR/wot-thing-description11/#eventaffordance) also
describes optional event response, subscription and cancellation attributes. These are not used in HiveOT as subscription is not handled by a Thing but by the Hub message bus.

### The "properties" Event

In HiveOT, changes to property values are sent using events. Rather than sending a separate event for
each property, HiveOT defines a 'properties' event. This event contains a properties map with
property name-value pairs. The concern this tries to address is that this reduces the amount of
events that need to be sent by small devices, reducing battery power and bandwidth.

As the 'properties' event is part of the HiveOT standard, it does not have to be included in the 'events' section of the TDs (but is recommended).  (should this be part of a HiveOT @content? tbd)

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

For example, when a name property has changed the event looks like:

```
{
  "name": "new name",
}
```

## Actions

Actions are used to control inputs and change the value of configuration properties.

The format of actions is defined in the Thing Description document
through [action affordances](https://www.w3.org/TR/wot-thing-description/#actionaffordance).

Similar to events, the actionName is the instance name of the action. By default this is the action type name defined in the vocabulary in the PropName section. In case the device supports multiple actions of the same type, like a multi-button switch, the action name contains the instance ID and the '@type' attribute contains the action type name.

```
{
  "actions": {
    "{actionName}": ActionAffordance,
    ...
  }
}
```

Where {actionName} is the instance name of the action as defined in the vocabulary, and ActionAffordance describes the action details. The action name shares the namespace with events and properties. The result of an action can be notified using an event with the same name and shown with a property of the same name:

```
{
  ...interactionAffordance,
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
    "onoff": {
      "title": "Control the on or off status of a switch",
      "input": {
        "type": "boolean",
        "title": "true/false for on/off"
      }
    }
  }
}
```

The action message to turn the switch on then looks like this:

```json
{
  "value": true
}
```

### The 'properties' Action

Similar to a properties Event, HiveOT standardizes a "properties" action. To change a configuration value, a properties action must be submitted containing a map of requested property name and value pairs. No additional action affordance is needed to write properties although this is recommended.

For example, when the Thing configuration property called "alarmThreshold" changes, the action looks
like this.

```json
{
  "alarmThreshold": 25
}
```

## Links

The spec describes a [link](https://www.w3.org/TR/wot-thing-description11/#link) as "A link can be
viewed as a statement of the form "link context has a relation type resource at link target", where
the optional target attributes may further describe the resource"

In HiveOT a link can be used as long as it is not served by the IoT device, as this would conflict
with the paradigm that "Things are not servers".

## Forms

The WoT specification for a [Form](https://www.w3.org/TR/wot-thing-description11/#form) says: "A form
can be viewed as a statement of "To perform an operation type operation on form context, make a
request method request to submission target" where the optional form fields may further describe the
required request."

As HiveOT does not allow direct protocol access to Things, Forms are ignored in published TDs. The Hub might instead replace the TD Forms section with a FOrm describing the Hub protocol to interact with the device via the Hub.

### SecuritySchema 'scheme' (1)

In HiveOT all authentication and authorization is handled by the Hub. Therefore, the security scheme
section only applies to Hub interaction and does not apply to HiveOT Things. The Hub service used to interact with Things will publish a TD that includes a SecuritySchema needed to interaction with the Hub.

# REST APIs

HiveOT compliant Things do not implement TCP/Web servers. All interaction takes place via HiveOT Hub services
and message bus. Therefore, this section only applies to Hub services that provide a web API. Instead the Hub Gateway Service provides web REST API's.

Hub services that implement a REST API follows the approach as described in Mozilla's Web Thing REST
API](https://iot.mozilla.org/wot/#web-thing-rest-api).

```http
GET https://address:port/things/{thingID}[/...]
```

The WoT examples often assume or suggest that Things are directly accessed, which is not allowed in HiveOT. Therefore, the implementation of this API in HiveOT MUST follow the following
rules:

1. The Thing address is that of the hub it is connected to.
2. The full thing ID must be included in the API. The examples says 'lamp' where a Thing ID is a
   URN: "urn:device1:lamp" for example.
