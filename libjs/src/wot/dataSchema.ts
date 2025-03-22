// Definition of the DataSchema used in TD affordances

import {WoTDataTypeNone} from "../api/vocab/vocab.js"
import {PropertyAffordance} from "./TD.ts";
export default class DataSchema extends Object {
    public constructor(init?: Partial<DataSchema>) {
        super();
        Object.assign(this, init)
    }

    // Used to indicate input, output, attribute. See vocab.WoSTAtType
    public "@type": string | undefined = undefined

    // Provides a default value of any type as per data schema
    public default: string | undefined = undefined

    // Provides additional (human-readable) information based on a default language
    public description: string | undefined = undefined
    // Provides additional nulti-language information
    public descriptions: string[] | undefined = undefined

    // Restricted set of values provided as an array.
    //  for example: ["option1", "option2"]
    public enum: any[] | undefined = undefined

    // number maximum value
    public maximum: number | undefined = undefined

    // maximum nr of items in array
    public maxItems: number | undefined = undefined

    // string maximum length
    public maxLength: number | undefined = undefined

    // number minimum value
    public minimum: number | undefined = undefined

    // minimum nr of items in array
    public minItems: number | undefined = undefined

    // string minimum length
    public minLength: number | undefined = undefined

    // property map when type is object
    public properties: { [key: string]: Partial<PropertyAffordance> } |undefined= undefined;

    // Optional nested properties. Map with PropertyAffordance
    // used when a property has multiple instances, each with their own name
    // public properties: Map<string, Partial<PropertyAffordance>> | undefined = undefined

    // Boolean value to indicate whether a property interaction / value is read-only (=true) or not (=false)
    // the value true implies read-only.
    public readOnly: boolean = true

    // Human-readable title in the default language
    public title: string | undefined
    // Human-readable titles in additional languages
    public titles: string[] | undefined = undefined

    // Type provides JSON based data type,  one of DataTypeNumber, ...object, array, string, integer, boolean or null
    public type: string = WoTDataTypeNone

    // See vocab UnitNameXyz for units in the WoST vocabulary
    public unit: string | undefined = undefined

    // Boolean value to indicate whether a property interaction / value is write-only (=true) or not (=false)
    public writeOnly: boolean = false

    // Enumeration table to lookup the value or key
    // FIXME: use oneOf object
    private enumTable: Object | undefined = undefined

    // Change the property into a writable configuration
    public SetAsConfiguration(): DataSchema {
        this.readOnly = false
        return this
    }

    // Add a list of enumerations to the schema.
    // This changes the schema to DataTypeString, fills in the enum array of strings, and
    // sets initialValue to the enum name.
    //
    // @param enumeration is a map from enum values to names and vice-versa
    // @param initialValue is converted to name and stored in the schema as initialValue (for testing/debugging) 
    public SetAsEnum(enumeration: Object): DataSchema {
        // FIXME: use oneOf object - match golang
        this.enumTable = enumeration
        let keys = Object.values(enumeration)
        this.enum = keys.filter((key: any) => {
                let isName = (!Number.isFinite(key))
                return isName
            }
        )
        return this
    }

    // Set the description and return this
    public  SetDescription(description: string): DataSchema {
        this.description = description
        return this
    }
}

