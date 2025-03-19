# ZwaveJS binding for HiveOT - Design

## Introduction

The binding is based on the excellent [ZWaveJS library](https://zwave-js.github.io/node-zwave-js/#/) for accessing ZWave USB control sticks. 
It builds WOT Thing Description documents using the zwave-js ZWaveNodes and publishes these and node
events to the Hub pubsub message bus. Action requests received from the pubsub message bus are passed on to the
corresponding ZWave node.

## Communication Protocols

This binding uses one of the HiveOT protocols available on JS. Currently this is the HiveOT Http/SSE-SC transport protocol. 

Javascript transport bindings using Websocket and MQTT are planned.

# Mapping ZWave to HiveOT

ZWaveJS uses 'Value IDs', containing command class, propertyName and propertyKey and CC metadata to define type and capabilities of ZWave devices. HiveOT uses the WoT TD standard to describe IoT devices using properties (attributes and configuration), actions (inputs) and events (outputs). How are value IDs mapped to the WoT TD? Read on for the answer to this exciting question.

One problem is that WoT does not distinguish between attributes and device sensor values, nor between writable attributes (configuration) and device actuator values.

This binding follows the hiveot approach that internal state are properties and external state changes are reported using events. Actuators are controlled through actions and report their current state as properties. 


## Mapping of Value ID to Property, Event and Action keys

The TD document contains three maps: properties, events and actions. The keys of these maps are unique property instance IDs that are used when sending events and receiving actions. These IDs are not for human presentation but must be immutable within the device.

The property, event and action IDs are constructed from: VID property + propertyKey + endpoint, where propertyKey is only used when it is defined, and endpoint is only used when multiple instances exist.

For example, a scene controller has a VID with property 'scene' and propertyKey '001' for the first scene. When a scene key is touched, the event will have a key of 'scene-001'.

Clients will receive a TD and events with this event key. In order to automatically understand the meaning of this event, the @type holds the vocabulary defined term for this event. In this case 'scene'.

...but wait, there is more...

## Mapping of Value ID to Property, Event or Action

VIDs need to be mapped to a TD property, event or an action. How to decide?

ZWave Command Classes are used to decide if a VID is a property, action or event. CC's for actuator devices are actions, while CC's for data reporting devices are events. The remainder are properties.

An override map can override of the general rules for each ValueID and change whether it is a property, action or event, name the property type, and data type.

...but wait, there is even more...

## Actuator State

In ZWave, actuator status is reported through its the Basic CC 'currentValue' attribute, while to control the actuator the Basic CC 'targetValue' attribute should be used. Note that this is zwave specific and the relationship between these is not obvious to consumers. 

In WoT this problem endures unfortunately as there is no way to express a relationship between an actuator and its state. One possible workaround is to control actuators through writable properties. However, this would hide the important distinction between configuration properties and actuator controls. HiveOT chooses a different approach using actions and properties.

HiveOT presents actuator state through a read-only property and actuator control through an action with the same name. If a read-only property has a corresponding action then this is by definition an actuator whose state is presented through a property and controlled through the action.

For actuators the action output dataschema is not used and ignored as the corresponding property already contains this information.

If an actuator affects the state of multiple properties then this can be presented as an object with a nested property for each affected state. The actuator output dataschema reflects this as well. This should be avoided if possible as presentation is more difficult. 

The zwave-js binding presents the 'currentValue' attribute as the read-only 'value' property. The corresponding 'value' action and controls the actuator through the internal 'targetValue' attribute.


## Mapping of Vocabulary Using @type Fields

Information is easiest consumed if the terminology used is consistent among various data sources.

Different technologies and device manufacturers however can use different terminology to indicate the same thing. Some might use 'temp', 'temperature' or 'degrees' for example. HiveOT attempts to adhere to a common ontology, but this seems to be a rather complex topic. See for example W3C's sematic sensor network ontology : https://www.w3.org/TR/vocab-ssn/#intro. Without a well known common vocabulary for IoT data, HiveOT vocabulary is based on terminology used in ZWave, Zigbee, Ocap, and other automation solutions. Units are based on SI metric units.

Since this ~~mess~~ ambiguity is not likely to be solved anytime soon we do what is commonly done in this situation. We add to the confusion with another vocabulary.

Therefore, HiveOT defines its vocabulary. There are defined in a capnp file that can be compiled to various programming languages. This vocabulary is based on commonly used terminology and not limited to one specific ontology. Each binding must map the names used in
their specific technology to this vocabulary. This binding uses a ~~versioned~~ vocabulary table that can easily be updated and corrected.

So, how is this helping the end-user? Each property, event and action definition has a '@type' field with the standardized vocabulary name of the property/event/action. A user interface can present the human name based on the vocabulary if the @type field is used. Automation rules can use it without needing to consider the various terms used by zwave, one-wire, ocap and so on.

Note that this section is fluid. The W3C WoT definition of this field defines that it is tied to a JSON-LD schema in the @context section of the TD. In this case the @type field is prefixed with the schema name and a colon.
Well that is the plan in theory. Currently, the names are plain vocabulary words. If a really nice vocabulary is found, then future iterations might prefix it using it as a schema.

Currently, this binding uses a copy of the capnproto defined vocabulary implemented in Typescript, as capnproto compilation to javascript is ~~a PITA~~ not yet production ready.

...still here? wow...



## Title and Description

The WoT TD supports language translatable names for each property, event and action. The 'title' and 'description' fields are intended to be read by humans.

The ZWaveJS VID metadata 'label' field is used as the 'title' while the and optional a 'description' field is used as the 'description'. If no description is available, the Command Class name and label are used as description.

In short, for presentation use the title and description fields, for automation use the @type field, if available.