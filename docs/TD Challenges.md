# Questions related to the use of TD in hiveot

1. How to define an event or action in the TD that carries a TD? [Solved]
> Use-case: Thing agents (like protocol bindings) publish events with the TD of the things they manage.
> Use-case: Consumers query TD's from the directory.
* Solution: use the JSON serialized TD.

2. How to notify consumers of changed property values? [Solved]
> Use-case: When one or more property values have changed, consumers must be notified.
* Sending them as an event would mean duplicating all properties in the TD as events which seems overkill and not the intended use.
* Old solution: define a '$properties' event that contains a list of property values, similar to [webthings.io events resource](https://webthings.io/api/#events-resource).
* New solution: Use the WoT 'observeproperty' operation and let the protocol handle it.

3. How to write multiple properties? [solved]
> Use-case: user applies changes to multiple properties values in one request.
> Solution: Don't support writing multiple properties. Use writeproperty instead. This avoids the ambiguity of the payload. Writing properties is rare enough so it isn't needed.

4. How to identify the actual "meaning" of a property/event/action? [ambiguous, unsolved]
> How does the consumer know a property or event holds a temperature?
> How to standardize this between multiple device manufacturers?
*  Current solution: define a "hiveot:" namespace for the project in the context and use @type to specify the vocabulary, eg: "hiveot:temperature"
* This doesn't allow multiple manufacturers though and isn't really standard.
* Ideal solution: WoT defines an IoT vocabulary using ISO standards, that manufacturers follow. Better to have an 80% standardized vocabulary than none at all. This still needs the use of @type or something similar for identification.

5. How does discovery describe the place where Things are kept? [Answered]
* Answer: Ege sent a [link](https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec) to the discovery specification. To be further investigated.

6. Is there an implied or intended relationship between properties, events/actions, or should they be considered independent? [Answered]
> Use-case 1: send an event if one or more properties change.
* option1: send an event with the property value
> Use-case 2: send an event if an action has completed to notify other consumers.
* Current solution: property, events and actions with the same name refer to the same Thing state. 
* Answer: Officially, there is no relationship between properties, actions and events names.
* HiveOT's approach:
  1. HiveOT agents use events to report sensor state changes and changes that do not affect the state of the device.
  2. HiveOT agents use actions to control actuators and for invoking stateless actions.
  3. Everything else is considered an attribute or configuration property. 
  4. The digital twin makes the current state of sensors and actuator available as read-only properties with the same name. This is for convenience of consumers that expects all state to exist as properties. 

7. How best to request reading the 'latest' value of an event? [Answered]
* Answer: If the event represents a state, read its corresponding property value.
* Answer: Support Thing Level operations 'readevent' and 'readallevents' to read the latest value of any event. This is provided by the digital twin service and not expected from Thing protocol bindings.

8. How to define global constants? [Workaround]
> Use-case: Various properties, events and actions use the same type of values. For example, unit names, on/off and state values, etc.
* How best to define these for use by multiple things?  
* This is probably a JSON-LD or vocabulary related question, but it is a common case that would be helpful to have an example of. How to do this for maximum interoperability?
* Current Solution: add a dataschema with single or enum constants to 'schemaDefinitions'
  The code generator creates the type and constants in the scope of the agent defining the thing.

9. Would it be out of scope to use a TD to define a RPC service API? [Non-WoT Workaround]
 * Hiveot provides services for the directory, value storage, history storage, authentication and others using a TD, and generates the API with documentation from it (experimental).
 * Answer 1: yes out of scope for WoT
 * Answer 2: Hiveot provides a 'Consumer' client that runs of top of the transport protocol bindings and correlate a response with a request using the 'correlationID' field contained in request and response messages.
  
10. 5.3.3.1 SecurityScheme  [Ambiguous]
> The forth paragraph: "Security schemes generally may require additional authentication parameters, such as a password or key. The location of this information is indicated by the value associated with the name in, often in combination with the value associated with name."
* Is this an example of security through obscurity?  

11. How to add a description to enum values? [Non-WoT workaround]
> Use-case: If an input has a restricted set of values, the consumer will have to select one of those values. Enum values however do not have a presentable title or description.
* How to present enum values?
* Current workaround: Don't use enum. Use oneOf instead which is an array of dataschema. Store the values in each entry in the 'const' field.

12. How to add a multi-line description? [Non-WoT workaround]
> Use-case: documenting properties, inputs or outputs can sometimes take more than a single line. However the TDD only accepts a single string and JSON doesn't do multi-line strings.
> Current Solution: Add support for a 'comments' field in DataSchema that is an array of strings. The tdd2go generator will generate multiple comment lines for this.

12. Does a propertyAffordance have a 'required' field? [Ambiguous]
> The example https://www.w3.org/TR/wot-scripting-api/#approaches-to-wot-application-development shows a 'required' field used in a property affordance. However, the [TD PropertyAffordance](https://www.w3.org/TR/wot-thing-description11/#propertyaffordance) does not define it. 
* Just an observation. A required boolean field in a dataschema from the example above is easier to use than a 'required' field in objectschema that contains the names of required fields in the object. Having both however makes it worse.

13. The PropertyAffordance relationship with dataschema is confusing [golang, hard to work with]
> Quote: Property instances are also instances of the class DataSchema. Therefore, it can contain the type, unit, readOnly and writeOnly members, among others."

What if a property is a number, or array. Is the property affordances inheritance tree depending on the type field? Eg the inheritance of a 'number' property:
> PropertyAffordance -> InteractionAffordance -> NumberSchema -> DataSchema
 
With an array property:
> PropertyAffordance -> InteractionAffordance -> ArraySchema -> DataSchema

* The problem is that this cannot be implemented in golang using data structs.
* Workaround: flatten the dataschema with fields for all types of data schema
  and let the 'type' field determine which fields are used.

14. Where does the encoding/decoding of input and output payloads take place? [Answered]
TD Forms define a "contentType" field that describes the encoding of the payload.

Answer: encoding is handled in the transport protocol. The forms in the TD contain the available transport protocols and its encoding, for every single property, event and action affordance.

15. How does the IoT device know what type the data is in? [Answered]
* Answer: When receiving actions or properties, agents expect to receive inputs in the same format as was defined in the TD. Consumers receiving events and property values rely on the TD dataschema to interpret the data format for presentation or analysis.

16. Is it correct that the consumer must convert complex data types? [golang, workaround] 
* ISSUE (golang): complex data types that are returned by actions (as defined in the TD) must be converted to the proper complex data type in the consumer. Unmarshalling to interface{} (golang) however returns a non compatible object as the unmarshaller doesn't know the actual data type. Thus, this doesn't work for goland.
* Workaround: Return the wire format and provide an unmarshal method to the client where the client can provide the expected data type.

17. How to include metadata (thingID, name, clientID) in SSE messages? [Non-WoT workaround]
* use-case: agent receives an action for a Thing via its SSE connection (agents connect to the Hub). The SSE data is the input data as per TDD, but how to convey the thingID, action name and correlationID?
* Workaround 1: Encapsulate the message in an envelope and use the additionalResponses field in a Form to define the envelope schema. However this has to be repeated for every single action/property/event which is *very* wordy.
* Workaround 2: Push this to the transport protocol. In case of SSE use the ID field: {thingID}/{name}/{correlationID}. (chosen workaround)
  * Problem: how can this be described in a Form? it can't.

18. How to describe a map of objects in the action output dataschema? [Workaround]
* Workaround: don't use maps, use arrays.
* Solution: Define the output as an object schema with a single property without a name. That property then defines a nested schema of the actual data.
* Recommendation: As this is very obfuscated and there is already an array type, it would be helpful to add a 'map' type. 

19. How to describe basic vs advanced properties, events and actions in the TD? [Ambiguous, unsolved]
Use case: human consumers might be interested in some properties but not all of them. The human consumer should be able to just look at the key properties without being overwhelmed with less important or more advanced ones. Some zwave devices have close to 100 properties of which only a handful are useful to the regular consumer. How to differentiate them?
    * Note: also check out https://webthings.io/schemas/
Options:
    * option1: use the hiveot vocabulary to indicate basic properties using @type and the hiveot @context vocabulary. 'sensor', 'actuator'
    * option2: use events for sensors and actions for actuators.
    * option3: add a custom field 'isSensor' and 'isActuator' to property affordance to differentiate configuration from sensor/actuator type properties
    * option4: start an initiative to combine all existing ontologies into a single world wide accepted ontology, vocabulary and classification of the majority of IoT devices with their properties, events and actions. Great idea, ... but erm, this looks like a lot of work, unless you enjoy herding cats as a hobby.
    * Option5: push the problem to the client. Maybe use @type to identify which properties are considered important. This requires standardization of property @type which doesn't exist. 
Current solution:
    * option 2: sensors publish as events and actuators are controlled with actions.

20. Can authorization rules (eg required roles) be applied to a TD? [not supported] 
Use case: Only show allowed actions and properties to a user based on their authorization. 
* This is not supported in WoT TD
* Option 1: Take a 'capabilities' approach. Split services in groups with each group allowing a role to use. For example, admin vs consumer roles. Each group is defined as a service with its own TD. The TD can include a custom field indicating which role a Thing allows.
* Option 2: The service programmatically tells the authz service which roles can use a Thing it publishes. An admin can potentially override this in the authz service.
* Option 3: Add custom 'allow' and 'deny' fields to each action that lists which roles are allowed/denied invoking an action.

21. How does a client know if an output or alternative response is received? [Ambiguity]
Use case: client invokes action and expects an output value. The TD describes a form with additionalResponses, for example to report an error. The client receives the response as per TD and needs to differentiate somehow between normal and additional responses. 
Section "5.3.4.2.2 Response-related Terms Usage" describes a response name-value pair that can be used, but where is it described? How does this fit in the response data? 
* Workaround 1: use one-of as the data schema and describe the output based on a flag. This leads to all action outputs to have a dataschema of type one-of. This defeats the purpose of additionalResponses as it doesn't use it.
* Workaround 2: parse the expected output and on failure parse using the schemas from additionalResponses. There is no way of telling which one to use until one fails. 'Try it until it works' is not a specification.
* Workaround 3: include a 'dataschema' metadata field in the transport that identifies the dataschema of the output.  

