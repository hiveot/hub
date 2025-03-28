# Questions related to the use of TD in HiveOT

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
* Answer: Ege sent a [link](https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec) to the discovery specification. Using DNS-SD publish a record with TD in the TXT field pointing to the TD of the directory service.

6. When to use a property, event, or action? [Answered,workaround]
There are some definitions in the wot-architecture but these leave some room for interpretation:
* Action: "... allows to invoke a function of the Thing, which manipulates state (e.g., toggling a lamp on or off) or triggers a process on the Thing"
* Events: "... describes an event source, which asynchronously pushes event data to Consumers (e.g., overheating alerts)."
* Property: "...exposes state of the Thing. This state can then be retrieved (read) and optionally updated (write). "

* How to decide if a property is writable?
* What constitutes an event source?
* When is something an action?

HiveOT Rules: 
    1. Event sources reflect changes to external state. For example, changes to sensors and inputs are sent as events. These include alarms that are the result of external state like a high or low temperature alarm. 
    2. Properties reflect internal state.
    3. Writable properties and actions are mutually exclusive. There is only one way to control a device. This avoids confusion of the user. 
    4. Writable properties are always idempotent. If it is writable but not idempotent then it is an action with a read-only property. For example 'increment dimmer' is an action.
    5. Actions are for device control by operators. Actuators like switches, valves and motors, are almost always controlled through actions. 
    6. If it is an action for an operator it is an action in the TD.
    7. State affected by an action has a corresponding read-only property with the same name and same data schema. 
        * This fills a gap in the specification to link the action output to the affected state.
        * This supports presenting the current status of an action even if manual control of an actuator has changed its state. (present the corresponding property)
        * If multiple states are affected by an action then the output/property schema is an object containing each affected state.

7. Is there an implied or intended relationship between properties/actions, or should they be considered independent? [Answered,workaround]
> Use-case: an action affects state that changes one or more properties.
* Current solution: properties and actions with the same name refer to the same Thing state. 
* Answer: Officially, there is no relationship between properties, actions and events names.
* HiveOT's approach (filling in a gap in the spec):
  1. Devices/services define properties for all state. 
  2. Agents use actions to control actuators and define read-only properties for the affected state using the same name as the action affordance.
  3. Actions output matches the corresponding property schema.
  4. The property contains the last value.
  5. queryaction returns the last action issued. This can differ from the property value if the device can be activated manually. 

8. How best to request reading the 'latest' value of an event? [Answered]
* Answer 1: When subscribing to events, receive the latest event value for each event.
* Answer 2: Support Thing Level operations 'readevent' and 'readallevents' to read the latest value of any event. This is provided by the digital twin service and not expected from Thing protocol bindings.

9. How to define global constants? [Workaround]
> Use-case: Various properties, events and actions use the same type of values. For example, unit names, on/off and state values, etc.
* How best to define these for use by multiple things?  
* This is probably a JSON-LD or vocabulary related question, but it is a common case that would be helpful to have an example of. How to do this for maximum interoperability?
* Current Solution: add a dataschema with single or enum constants to 'schemaDefinitions'
  The code generator creates the type and constants in the scope of the agent defining the thing.

11. 5.3.3.1 SecurityScheme  [Ambiguous]
> The forth paragraph: "Security schemes generally may require additional authentication parameters, such as a password or key. The location of this information is indicated by the value associated with the name in, often in combination with the value associated with name."
* Is this an example of security through obscurity?  

12. How to add a description to enum values? [Non-WoT workaround]
> Use-case: If an input has a restricted set of values, the consumer will have to select one of those values. Enum values however do not have a presentable title or description.
* How to present enum values?
* Current workaround: Don't use enum. Use oneOf instead which is an array of dataschema. Store the values in each entry in the 'const' field.

13. How to add a multi-line description? [Non-WoT workaround]
> Use-case: documenting properties, inputs or outputs can sometimes take more than a single line. However the TDD only accepts a single string and JSON doesn't do multi-line strings.
> Current Solution: Add support for a 'comments' field in DataSchema that is an array of strings. The tdd2go generator will generate multiple comment lines for this.

14. Does a propertyAffordance have a 'required' field? [Ambiguous]
> The example https://www.w3.org/TR/wot-scripting-api/#approaches-to-wot-application-development shows a 'required' field used in a property affordance. However, the [TD PropertyAffordance](https://www.w3.org/TR/wot-thing-description11/#propertyaffordance) does not define it. 
* Just an observation. A required boolean field in a dataschema from the example above is easier to use than a 'required' field in objectschema that contains the names of required fields in the object. Having both however makes it worse.

15. The PropertyAffordance relationship with dataschema is confusing [golang, hard to work with]
> Quote: Property instances are also instances of the class DataSchema. Therefore, it can contain the type, unit, readOnly and writeOnly members, among others."

What if a property is a number, or array. Is the property affordances inheritance tree depending on the type field? Eg the inheritance of a 'number' property:
> PropertyAffordance -> InteractionAffordance -> NumberSchema -> DataSchema
 
With an array property:
> PropertyAffordance -> InteractionAffordance -> ArraySchema -> DataSchema

* The problem is that this cannot be implemented in golang using data structs.
* Workaround: flatten the dataschema with fields for all types of data schema
  and let the 'type' field determine which fields are used.

16. Where does the encoding/decoding of input and output payloads take place? [Answered]
TD Forms define a "contentType" field that describes the encoding of the payload.

Answer: encoding is handled in the transport protocol. The forms in the TD contain the available transport protocols and its encoding, for every single property, event and action affordance.

17. How to include metadata (thingID, name, clientID) in SSE messages? [Non-WoT workaround]
* use-case: agent receives an action for a Thing via its SSE connection (agents connect to the Hub). The SSE data is the input data as per TDD, but how to convey the thingID, action name and correlationID?
* Workaround 1: Encapsulate the message in an envelope and use the additionalResponses field in a Form to define the envelope schema. However this has to be repeated for every single action/property/event which is *very* wordy.
* Workaround 2: Push this to the transport protocol. In case of SSE use the ID field: {thingID}/{name}/{correlationID}. (chosen workaround)
  * Problem: how can this be described in a Form? it can't.
* Workaround 3: Push this to the transport protocol. In case of SSE use a RequestMessage/ResponseMessage/NotificationMessage envelope as the payload instead of the raw data. This is effectively a different protocol binding.  

18. How to describe a map of objects in the action output dataschema? [Workaround]
* Workaround: don't use maps, use arrays.
* Solution: Define the output as an object schema with a single property without a name. That property then defines a nested schema of the actual data.
* Recommendation: As this is very obfuscated and there is already an array type, it would be helpful to add a 'map' type.
* Recommendation: If a 'map' type cannot be added then please document this in the DataSchema examples. 

19. How to describe basic vs advanced properties, events and actions in the TD? [Ambiguous, unsolved]
Use case: human consumers might be interested in some properties but not all of them. The human consumer should be able to just look at the key properties without being overwhelmed with less important or more advanced ones. Some zwave devices have over 100 properties of which only a handful are useful to the regular consumer. How to differentiate them?
    * Note: also check out https://webthings.io/schemas/
Options:
    * option1: use the hiveot vocabulary for classification of properties using @type and the hiveot @context vocabulary. 'sensor', 'actuator', ...
    * option2: add a custom field 'isSensor' and 'isActuator' to property affordance to differentiate configuration from sensor/actuator type properties
    * option3: start an initiative to combine all existing ontologies into a single world wide accepted ontology, vocabulary and classification of the majority of IoT devices with their properties, events and actions. Great idea, ... but erm, this looks like a lot of work, unless you enjoy herding cats as a hobby.
    * Option4: push the problem to the client. Maybe use @type to identify which properties are considered important. This requires standardization of property @type which doesn't exist. 

Current solution:
    * option 1: Using @type but a common classification is needed.

20. Can authorization rules (like required client roles) be applied to a TD? [not supported] 
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

22. What to use to present a device title?
Use case: a device title is not always a good human identifier. Therefore it should be editable. The TD title is not editable and contains a read-only technical name. How to edit the title?
* Workaround: The digital twin includes a title property in all TDs that is configurable. The UI uses this property as the device title. If no such property exists (non-hiveot device) then fall back to the TD title. If the device has a title property already then it is used instead.
* The same goes for description.

23. How to re-use a data schema in multiple affordances? [Incompatible Workaround]

Workaround: Define them in the TD 'schemaDefinitions' section and use them as a data type in the data schema. Yeah this is not WoT compatible.

24. How to re-use a data schema from one TD in another? [Not solved]
Currently a data type has to be redefined in each TD. 
Can a 'link be used here?'

Proposed: Use a Link to point to the dataschema of another TD.

25. Forms are overkill, complex, unnecessary, and very very annoying. 
Get rid of forms. 

All protocols need to exchange the same information: operation, thingID, affordance name, correlationID, messageID, message type (request, response, notification), action status, and input or output data. 

Thats it. It always boils down to this set of information. Why on earth would you want to regurgitate this in all possible variations for every affordance and every protocol throughout the TD? They can just be variables in a protocol message.

Having to specify a form in every single affordance can double the TD size, cost extra processing power and resources, without any benefit. It adds unnecessary complexity, increases the risk of bugs, makes testing more difficult and hinders adoption. I almost threw in the WoT towel with hiveot due to forms. In the end I chose to ignore them except for the Thing level forms. (yeah too bad so sad, use my client library). It is the wrong solution for a simple problem. Lets get past it in TD2.0 while we can. 

Proposal: 
Leverage operations. Let the protocol binding decide how to transfer the information for an operation. The TD only needs to define the protocol used and how to establish a secure connection. A Thing level protocols section in the TD is all that is needed. For example:
```
{
  protocols: {
      "websocket": {
         "endpoint": "wss://address:port/path",
         "security": "stuff goes here",
         "encoding": "json",
      }
      "mqtt": {
         "endpoint": "mqtts://address:port/"
         "security": "stuff goes here",
         ...
      }
  }
}
```

The TD defines how to connect. The protocol defines how to exchange messages for operations. 

All protocols exchange the same information: operation, thingID, affordance name, correlationID, messageID, message type (request, response, notification) and input or output data. Let the protocol specify how to exchange this information.

The latest websocket proposal (not strawman) is a good example. It works.   

As a bonus this is an oppertunity to standardize the messaging envelopes used to send requests, receive responses and notifications instead of reinventing this wheel for each protocol. This would reduce SDK development time, simplify a standard test suite to validate interoperability, and also encourage adoption especially for consumers that need to support multiple protocols. A good candidate for standardized message envelopes are the ones defined in the latest websocket protocol (not the old strawman).

This is IoT, not a multi-media server chat, telephony, broadcasting system.

In case it is really necessary to change the encoding for a specific affordance then specify the different encoding in the data schema of that affordance.

What is lost with this change is the ability to use different protocols for different affordances of the same Thing. I have yet to see a valid use-case for this that can't be addressed with the proposed solution. 
