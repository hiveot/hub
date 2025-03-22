// Definition of the Thing's TD, Thing Description document
// This consists of the TD itself with properties

import DataSchema from "./dataSchema.ts";

const TD_CONTEXT = "https://www.w3.org/2022/wot/td/v1.1"
const HT_CONTEXT = "https://www.hiveot.net/vocab/v0.1"

export class InteractionAffordance extends Object {
    // Unique name of the affordance, eg: property, event or action name
    // While not part of the official specification, it allows passing the affordance
    // without having to separately pass its id.
    id: string = ""

    // type of affordance, eg temperature, switch,...
    "@type": string | undefined

    public comment: string|undefined

    // Provides additional (human-readable) information based on a default language
    public description: string | undefined
    // Provides additional nulti-language information
    public descriptions: string[] | undefined = undefined

    // Human-readable title in the default language
    public title: string | undefined
    // Human-readable titles in additional languages
    public titles: string[] | undefined = undefined

}

/** Thing Description Action Affordance
 */
export class ActionAffordance extends InteractionAffordance {
    /**
     * Input data for the action when applicable
     */
    public input?: DataSchema = undefined
    /**
     * Output data for the action when applicable
     */
    public output?: DataSchema = undefined

    /**
     * Action is idempotent. Repeated calls have the same result.
     */
    public idempotent?: boolean = undefined

    // // action input parameters
    // public inputs = new Map<string, {
    //   WoTDataType?: string,
    //   WoTProperties?: Map<string, string>,
    //   WoTRequired?: boolean,
    // }>()

    // Create an action affordance instance with a schema for its input, if any
    public constructor(dataSchema?: DataSchema) {
        super();
        this.input = dataSchema
    }

    // set the action's vocabulary @type field
    public setVocabType(atType: string): ActionAffordance {
        this["@type"] = atType
        return this
    }
}

/** Thing Description Event Affordance
 */
export class EventAffordance extends InteractionAffordance {
    // Data schema of the event instance message, eg the event payload
    public data?: DataSchema

    // Create an event affordance instance with a schema for its data, if any
    constructor(dataSchema?: DataSchema) {
        super();
        this.data = dataSchema
    }

    // set the event's vocabulary @type field
    public setVocabType(atType: string): EventAffordance {
        this["@type"] = atType
        return this
    }
}

/** Thing Description property affordance
 * The specification says this is an interaction affordance that is also a data schema?
 * JS doesn't support multiple inheritance so we'll use a dataschema and add the missing
 * 'forms' field from the interaction affordance.
 */
export class PropertyAffordance extends DataSchema {

    // name is the property name in the map, so it is available when the properties are provided as an array
    name: string = ""
    //
    // // Optional nested properties. Map with PropertyAffordance
    // // used when a property has multiple instances, each with their own name
    // public properties: Map<string, PropertyAffordance> | undefined = undefined

    // set the property's vocabulary @type field
    public setVocabType(atType: string): PropertyAffordance {
        this["@type"] = atType
        return this
    }
}


// Form that describes an operation on a thing
export class TDForm {
    public operation: string = ""
    public href: string = ""
    public subprotocol : string|undefined

    // return the href in this form
    public getHRef() {return this.href}
}

/** Thing description document
 */
export default class TD extends Object {

    /**
     * Create a new instance of Thing Description document
     *
     * @param deviceID thingID of this device
     * @param title human readable title (name) of the device
     * @param deviceType one of vocabulary's DeviceTypeXyz
     * @param description more detailed description of the device
     */
    constructor(deviceID: string, deviceType: string, title: string, description: string) {
        super();
        this.id = deviceID;
        this["@context"] = [
            TD_CONTEXT,
            { "hiveot": HT_CONTEXT }
        ]
        this["@type"] = deviceType;
        this.title = title;
        this.description = description;
        this.created = new Date().toISOString();
        this.modified = this.created;
    }
    /** JSON-LD context */
    public "@context": any[] = [];

    /** Type of thing defined in the vocabulary */
    public "@type": string = "";

    /** Unique thing ID */
    public readonly id: string = "";

    /** Document creation date in ISO8601 */
    public created: string = "";

    /** Document modification date in ISO8601 */
    public modified: string = "";

    /** Human description for a thing */
    public description: string = "";

    /** Human-readable title for ui representation */
    public title: string = "";

    public forms: TDForm[]|undefined

    /**
     * Collection of properties of a thing
     * @param key see WoST vocabulary PropNameXxx
     */
    public readonly properties: { [key: string]: PropertyAffordance } = {};

    /** Collection of actions of a thing */
    public readonly actions: { [key: string]: ActionAffordance } = {};

    /** Collection of events (outputs) of a thing */
    public readonly events: { [key: string]: EventAffordance } = {};


    // AddAction provides a simple way to add an action to the TD
    // This returns the action affordance that can be augmented/modified directly
    //
    // If the action accepts input parameters then set the .Data field to a DataSchema instance that
    // describes the parameter(s).
    // If the action has a vocabulary type then set .actionType to it
    //
    // @param id is the key under which it is stored in the action map.
    // @param title is the short display title of the action.
    // @param description optional detailed description of the action
    // @param input with optional dataschema of the action input data
    AddAction(id: string, title: string, description?: string, input?: DataSchema): ActionAffordance {
        const action = new ActionAffordance()
        action.id = id;
        action.title = title
        action.description = description
        action.input = input
        this.actions[id] = action;
        return action
    }

    // AddEvent provides a simple way to add an event definition to the TD.
    // This returns the event affordance that can be augmented/modified directly.
    //
    // To set a known vocabulary @type, use setVocabType on the result
    //
    // @param id is the event instance ID under which it is stored in the event map.
    //        This can be anything arbitrary as long as the TD and value event use the same ID.
    // @param title is the short display title of the action.
    // @param description optional detailed description of the action
    // @param dataSchema optional event data schema
    AddEvent(id: string,  title: string, description?: string, dataSchema?: DataSchema): EventAffordance {
        const ev = new EventAffordance()
        ev.id = id;
        ev.title = title ? title : id;
        ev.description = description
        ev.data = dataSchema
        this.events[id] = ev;
        return ev
    }

    // AddProperty provides a simple way to add a Thing property to the TD
    // This returns the property affordance that can be augmented/modified directly
    // By default this property is read-only. (eg an attribute)
    //
    // @param name is the instance name under which it is stored in the property affordance map.
    // @param title is the title used in the property.
    // @param description of the property
    // @param dataType is the type of data the property holds, DataTypeNumber, ..Object, ..Array, ..String, ..Integer, ..Boolean or null
    // @param vocabType is the vocabulary type of this property, if known
    AddProperty(name: string, title: string,  description:string,dataType: string, vocabType?:string): PropertyAffordance {
        const prop = new PropertyAffordance()
        prop.name = name;
        prop.type = dataType;
        prop.title = title ? title : name;
        prop.readOnly = true;
        prop.description = description
        if (vocabType) {
            prop.setVocabType(vocabType)
        }
        this.properties[name] = prop;
        return prop
    }


    // AddPropertyIf only adds the property if the first parameter is not undefined 
    //
    // @param initialValue add the attribute if the initial value is not undefined
    // @param name is the instance name under which it is stored in the property affordance map.
    // @param title is the title used in the property. Leave empty to use the name.
    // @param description optional
    // @param dataType is the type of data the property holds, DataTypeNumber, ..Object, ..Array, ..String, ..Integer, ..Boolean or null
    // @param vocabType optional vocabulary @type field
    AddPropertyIf(initialValue: any, name: string,
                  title: string, description:string,
                  dataType: string, vocabType?:string): PropertyAffordance | undefined {

        if (initialValue != undefined) {
            const prop =  this.AddProperty(name, title, description,dataType)
            if (vocabType){
                prop.setVocabType(vocabType)
            }
            return prop
        }
        return undefined
    }


    // Convert the actions map into an array for display
    public static GetThingActions = (td: TD): Array<ActionAffordance> => {
        const res = new Array<ActionAffordance>()
        if (!!td && !!td.actions) {
            for (const [_key, val] of Object.entries(td.actions)) {
                res.push(val)
            }
        }
        return res
    }


    // Convert readonly properties into an array for display
    // Returns table of {name, tdproperty}
    public static GetAttributeNames = (td: TD): string[] => {
        const res = Array<string>()
        if (!!td && !!td.properties) {
            for (const [key, val] of Object.entries(td.properties)) {
                if (val.readOnly) {
                    res.push(key)
                }
            }
        }
        return res
    }


    // Returns names of configuration properties
    public static GetConfigurationNames = (td: TD): string[] => {
        const res = Array<string>()
        if (!!td && !!td.properties) {
            for (const [key, val] of Object.entries(td.properties)) {
                if (!val.readOnly) {
                    res.push(key)
                }
            }
        }
        return res
    }

    public static GetThingEvents = (td: TD): Array<EventAffordance> => {
        const res = Array<EventAffordance>()
        if (!!td && !!td.events) {
            for (const [_key, val] of Object.entries(td.events)) {
                res.push(val)
            }
        }
        return res
    }


    // Return the TD property with the given ID
    public static GetThingProperty = (td: TD, propID: string): PropertyAffordance | undefined => {
        let tdProp: PropertyAffordance | undefined = undefined
        if (!!td && !!td.properties) {
            tdProp = td.properties[propID]
        }
        return tdProp
    }

}

