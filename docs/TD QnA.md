# Questions related to the use of TD in hiveot

1. How to define an event or action in the TD that carries a TD?
> Use-case: Thing agents (like protocol bindings) publish events with the TD of the things they manage.
> Use-case: Consumers query TDD's from the directory.

* Current solution: return a string with the serialized TD.
* Preferred solution: reference a TD type for the return value.

2. How to notify consumers of changed property values?
> Use-case: When one or more property values have changed, consumers must be notified.
* Sending them as an event would mean duplicating all properties in the TD as events which seems overkill and not the intended use.
* Current solution: define a '$properties' event that contains a list of property values, similar to [webthings.io events resource](https://webthings.io/api/#events-resource).
* Current solution: send an event with the property name without having to define an event for it.

3. How to updates multiple properties? (SOLVED)
>  Use-case: user applies changes to multiple properties values in one request.
* Current solution: Similar to pt.2, define a "$properties" action that holds one or more property values.
* Answer: The TD form defines "writemultipleproperties" and "readallproperties" operation type. - investigate further

4. How to identify the "meaning" of a property/event/action?
> How does the consumer know a property or event holds a temperature?
> How to standardize this between multiple device manufacturers?
*  Current solution: define a "ht:" namespace for the project in the context and use @type to specify the vocabulary, eg: "ht:temperature"
* This doesn't allow multiple manufacturers though and isn't really standard.
* Ideal solution: WoT defines an IoT vocabulary using ISO standards, that manufacturers follow. Better to have an 80% standardized vocabulary than none at all. This still needs the use of @type or something similar for identification.

5. How does discovery describe the place where things are kept?
* Note: Ege sent a [link](https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec). To be further investigated.

6. Is there an implied or intended relationship between properties, events/actions, or should they be considered independent?
> Use-case 1: send an event if one or more properties change.
* option1: send an event with the property value
> Use-case 2: send an event if an action has completed to notify other consumers.
* Current solution: events and actions keys refer to the same Thing state. When an action completes, an event with the same name as the action notifies of this. The definition of action and property affordances imply a corresponding event. 

7. How best to request reading the 'latest' value of an event?
Can this be defined in a top level form in the TD as an expansion of 'readproperty'? or does this belong in the outbox/history services TD.   

8. How to define global constants?
> Use-case: Various properties, events and actions use the same type of values. For example, unit names, on/off and state values, etc.
* How best to define these for use by multiple things?  
* This is probably a JSON-LD or vocabulary related question, but it is a common case that would be helpful to have an example of. How to do this for maximum interoperability?
* Current Solution: add a dataschema with single or enum constants to 'schemaDefinitions'
  The code generator creates the type and constants in the scope of the agent defining the thing.

9. Would it be out of scope to use a TD to define a service API? Has this been done before?
 * The idea for hiveot is to define the directory, value storage, history storage, authentication and other services using a TD and generate the API with documentation from it.  
 * Current Solution: All services are defined through a TDD and tdd2go will generate 
   golang code for it.

  
10. 5.3.3.1 SecurityScheme
> The forth paragraph: "Security schemes generally may require additional authentication parameters, such as a password or key. The location of this information is indicated by the value associated with the name in, often in combination with the value associated with name."
   
* It would be nice if this made sense. So-far however, this has eluded me. It is quite ambiguous. When removing the second sentence, sense returns to some degree. If this is important however then please know that this important sentence is lost to this reader.  

11. How to add a description to enum values?
> Use-case: If an input has a restricted set of values, the consumer will have to select one of those values. Enum values however do not have a presentable title or description.
* How to present enum values?
* Current workaround: Don't use enum. Use oneOf instead which is an array of dataschema. Store the values in each entry in the 'const' field.

12. How to add a multi-line description?
> Use-case: documenting properties, inputs or outputs can sometimes take more than a single line. However the TDD only accepts a single string and JSON doesn't do multi-line strings.
> Current Solution: Add support for a 'comments' field in DataSchema that is an array of strings. The tdd2go generator will generate multiple comment lines for this.

12. Does a propertyAffordance have a required field?
> The example https://www.w3.org/TR/wot-scripting-api/#approaches-to-wot-application-development shows a 'required' field used in a property affordance. However, the [TD PropertyAffordance](https://www.w3.org/TR/wot-thing-description11/#propertyaffordance) does not define it. 
* Just an observation. A required boolean field in a dataschema from the example above is easier to use than a 'required' field in objectschema that contains the names of required fields in the object. Having both however makes it worse.

13. The PropertyAffordance relationship with dataschema is confusing
> Quote: Property instances are also instances of the class DataSchema. Therefore, it can contain the type, unit, readOnly and writeOnly members, among others."

What if a property is a number. Is the property affordances inheritance tree depending
on the data type? Eg the inheritance of a 'number' property:
> PropertyAffordance -> InteractionAffordance -> NumberSchema -> DataSchema
 
With an array property:
> PropertyAffordance -> InteractionAffordance -> ArraySchema -> DataSchema

* The problem is that this cannot be implemented in golang using data structs.
* Workaround: flatten the dataschema with fields for all types of data schema
  and let the 'type' field determine which fields are used.

14. Where does the encoding/decoding of input and output payloads take place?
TD Forms define a "contentType" field that describes the encoding of the payload.

Answer: encoding is handled in the transport protocol. The forms in the TD contain the available transport protocols and its encoding, for every single property, event and action affordance.

Note: The inbox, outbox, and history stores must handle their own encoding. This is decoupled from the transport protocols.

15. How does the IoT device know what type the data is in?
Answer: When receiving actions or properties, agents expect to receive inputs in the same format as was defined in the TD. Consumers receiving events and property values rely on the TD dataschema to interpret the data format for presentation or analysis.
Eg, pass an integer as a boolean or vice versa. The transport protocol decodes the data into its native format. When receiving events, clients therefore can only receive data with the 'any' type for the value produced by the unmarshaller. That means that when handling this, it must be cast to its expected native type before use. If the type doesn't match then this would have to fail gracefully.

16. Is it correct that the consumer must convert complex data types? 
ISSUE (golang): complex data types that are returned by actions (as defined in the TD) must be converted to the proper complex data type in the consumer. Unmarshalling to interface{} (golang) however returns a non compatible object as the unmarshaller doesn't know the actual data type. Thus, this doesn't work for goland.
Workaround: Return the wire format and provide an unmarshal method to the client where the client can provide the expected data type.

17. How to define the ThingMessage envelope in SSE messages?
When the server sends data to a client over SSE, it must include additional info such as senderID, thingID, affordance key and messageID. In http these can be put in the url and query parameters. With the SSE transport both the payload and metadata are embedded into a ThingMessage envelope. How to describe this?

18. How to describe a map of objects in the action output dataschema?

19. How to describe basic vs advanced properties, events and actions in the TD?
Use case: human consumers might be interested in some properties, events or actions but not all of them. The human consumer should be able to just look at the essential data without being overwhelmed with more advanced ones. Some zwave devices have close to 100 properties of which only a handful are useful to the regular consumer. How to differentiate them?
Note: check out https://webthings.io/schemas/
    * option1: use the ht vocabulary to indicate basic properties. This is not compatible with anything.
    * option2: add to the @context. Yeah ...
    * option3: start an initiative to combine all existing ontologies into a single world wide accepted ontology, vocabulary and classification of the majority of IoT devices with their properties, events and actions. This seems like a lot of work.

20. How to link an event received over SSE containing the result of an action to a corresponding action?
Use case: Consumer sends an action and expect its UI to show the progress and result of the action. 

21. Can authorization rules (eg required roles) be applied to a TD? 
Use case: Only show allowed actions to a user based on their authorization. 