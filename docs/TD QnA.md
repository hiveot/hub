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

3. How to updates multiple properties? (SOLVED)
>  Use-case: user applies changes to multiple properties values in one request.

* Current solution: Similar to pt.2, define a "$properties" action that holds one or more property values.
* Answer: The TD form defines "writemultipleproperties" and "readallproperties" operation type.

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
* option1: send an event with the property
* option2: is this where forms come in? 
> Use-case 2: send an event if an action has completed to notify other consumers.
* Current solution: events and actions keys refer to the same aspect. When an action completes, an event with the same key notifies of this. However, this seems to imply that all actions need a corresponding event defined. 

7. How best to request reading the 'latest' value vs historical values of an event?
TODO: check if forms accounts for this.

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
   
* It would be nice if this made sense. So-far however, sense has not visited this paragraph. It is quite ambiguous. When removing the second sentence, sense returns to some degree. If this is important however then please know that this important sentence is lost to this reader.  

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

Note1: Internally, the Hub transport client and server use an 'any' parameter for passing data. This is marshalled according to the transport protocol encoding, which is described in the TDD Forms.

Note2: The inbox, outbox, and history stores must handle their own encoding. This is decoupled from the transport protocols.

15. How does the IoT device know what type the data is in?
When receiving actions or properties, agents expect to receive inputs in the same format as was defined in the TD. Consumers receiving events and property values rely on the TD dataschema to interpret the data format for presentation or analysis.
Eg, pass an integer as a boolean or vice versa. The transport protocol decodes the data into its native format. When receiving events, clients therefore can only receive data with the 'any' type for the value produced by the unmarshaller. That means that when handling this, it must be cast to its expected native type before use. If the type doesn't match then this would have to fail gracefully.

16. Is it correct that the client must convert the data type? 
ISSUE: complex data types that are returned by actions must be converted to the proper complex data type. Unmarshalling to interface{} (golang) however returns a non compatible object as the unmarshaller doesn't know the actual data type. Thus, this doesn't work.
Workaround: Return the wire format and provide an unmarshal method to the client where the client can provide the expected data type.

17. How to define the ThingMessage wrapper in SSE messages?
When the server sends data to a client over SSE, it must include additional info such as senderID, thingID, affordance key and messageID. In http these can be put in the url and query parameters. With the SSE transport both the payload and metadata are embedded into a ThingMessage struct. How to describe this?

18. How to describe a map of objects in the action output dataschema?

19. How to describe basic vs advanced properties, events and actions in the TD?
    * option1: use the ht vocabulary to indicate basic properties
    * option2: add to the @context

20. How to link an event received over SSE containing the result of an action to a corresponding action?
