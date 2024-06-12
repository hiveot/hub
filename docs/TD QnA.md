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

* The problem is that I'm unable to implement this in a maintainable and easy to use fashion in golang.
* Workaround: flatten the property affordance with fields for all data schemas
  and let the 'type' field determine which fields are used.