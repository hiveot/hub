package gentypes

import (
	"fmt"

	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot/td"
	"golang.org/x/exp/slices"
)

func GenDataSchema(l *SL, schemaName string, ds *td.DataSchema) (err error) {
	schemaTypeName := ToTitle(schemaName)

	if ds.Type == "object" {
		err = GenDataSchemaObject(l, schemaTypeName, ds)
	} else if ds.Enum != nil {
		// define the agent scoped enum
		GenDataSchemaEnum(l, schemaTypeName, ds)
	} else if ds.OneOf != nil {
		// define the agent scoped onoff
		err = GenDataSchemaOneOf(l, schemaTypeName, ds)
	} else if ds.Default != "" {
		// define a agent scoped constant
		GenDataSchemaConst(l, schemaTypeName, ds)
	} else if ds.Const != nil {
		GenDataSchemaConst(l, schemaTypeName, ds)
	} else {
		// this is a native type. Nothing to do here.
		err = nil
	}
	return err
}

// GenDataSchemaObject generates a golang type of the given name for the dataschema.
//
// 1. If the dataschema has a 'schema' reference then use this as the type
// > type {typeName} {schemafield}
//
// 2. If the dataschema is a map (1 properties with "" key)
// > type {typeName} map[string]{propstype}
//
// 3. Default define a struct
//
//	> type {typeName} struct {
//	  ... properties
//	}
//
// data struct or map scoped to the agent from schema definition
func GenDataSchemaObject(l *SL, typeName string, ds *td.DataSchema) (err error) {
	l.Add("// %s defines a %s data schema.", typeName, ds.Title)
	GenDescription(l, ds.Description, ds.Comments)

	// 1. if ds.schema field is set then use it instead of a struct
	if ds.Schema != "" {
		l.Add("type %s %s", typeName, ds.Schema)
	} else if len(ds.Properties) == 1 && ds.Properties[""] != nil {
		// 2. if dataschema is a map
		mapSchema := ds.Properties[""]
		if mapSchema.Schema != "" {
			l.Add("type %s map[string]%s", typeName, mapSchema.Schema)
		} else {
			l.Add("type %s map[string]struct{", typeName)
			l.Indent++
			// define an agent wide data struct
			err = GenDataSchemaFields(l, typeName, mapSchema)
			l.Indent--

			l.Add("}")
		}
	} else {
		// 3. if dataschema is an object or array of objects
		// in both cases the content is a list of struct fields
		if ds.Type == "array" {
			l.Add("type %s []struct {", typeName)
			l.Indent++
			// NOTE: this currently only support a single array data type, similar to object
			err = GenDataSchemaFields(l, typeName, ds.ArrayItems)
			l.Indent--
		} else {
			l.Add("type %s struct {", typeName)

			l.Indent++
			// define an agent wide data struct
			err = GenDataSchemaFields(l, typeName, ds)
			l.Indent--
		}
		l.Add("}")
	}
	l.Add("")
	return err
}

// GenDataSchemaFields generates the fields of a golang struct from its DataSchema.
// Intended for generating fields in action, event, property affordances and schema definitions.
//
//	l is the output lines with generated source code
//	name is the field name of the dataschema if there is only a single field
//	ds is the dataschema to generate
func GenDataSchemaFields(l *SL, name string, ds *td.DataSchema) (err error) {
	// Each field of a dataschema is a native type or an object
	if len(ds.Properties) > 0 {
		err = GenSchemaAttr(l, ds.Properties)
	} else {
		// This dataschema has a single field
		props := map[string]*td.DataSchema{name: ds}
		err = GenSchemaAttr(l, props)
	}
	return err
}

// GenDescription generates the comments from description and comments
func GenDescription(l *SL, description string, comments []string) {
	if description != "" {
		l.Add("//")
		l.Add("// %s", description)
	}
	if comments != nil {
		//l.Add("//")
		for _, row := range comments {
			l.Add("// %s", row)
		}
	}
}

// GenSchemaAttr generates the attribute fields of a dataschema
//
//	attrMap contains a map of attribute name with their description
//
// func GenSchemaAttr(l *SL, attrList []SchemaAttr) {
func GenSchemaAttr(l *SL, attrMap map[string]*td.DataSchema) (err error) {

	names := utils.OrderedMapKeys(attrMap)
	for _, key := range names {
		attr := attrMap[key]
		keyTitle := ToTitle(key)
		goType := GoTypeFromSchema(attr)
		l.Add("")
		l.Add("// %s with %s", keyTitle, attr.Title)
		GenDescription(l, attr.Description, attr.Comments)
		if attr.Properties != nil {
			// nested struct
			err = GenSchemaAttr(l, attr.Properties)
			// } else if attr.Schema != "" {
			// field is a reference to a dataschema
			// typeName := ToTitle(attr.Schema)
			// l.Add("%s %s", keyTitle, typeName)
		} else if len(attr.AdditionalProperties) > 0 {
			// field is a map of dataschema
			//GenSchemaAttr(l, attr.AdditionalProperties)
		} else {
			isRequired := false
			if attr.Required != nil {
				isRequired = slices.Contains(attr.Required, key)
			}
			if !isRequired {
				// optional struct field is a pointer
				if attr.Schema != "" || goType == "struct" {
					l.Add("%s *%s `json:\"%s,omitempty\"`", keyTitle, goType, key)
				} else {
					l.Add("%s %s `json:\"%s,omitempty\"`", keyTitle, goType, key)
				}
			} else {
				// required field must be non-nil
				l.Add("%s %s `json:\"%s\"`", keyTitle, goType, key)
			}

		}
	}
	return err
}

// GenDataSchemaConst generates a constant value from schema definition
func GenDataSchemaConst(l *SL, title string, ds *td.DataSchema) {
	l.Add("")
	l.Add("// %s constant with %s", title, ds.Description)
	l.Add("const %s = %s", title, ds.Const)
}

// GenDataSchemaEnum generates enum type and constants from schema definition
func GenDataSchemaEnum(l *SL, enumTypeName string, ds *td.DataSchema) {

	goType := GoTypeFromSchema(ds)

	l.Add("// %s enumerator", enumTypeName)
	GenDescription(l, ds.Description, ds.Comments)
	l.Add("type %s %s", enumTypeName, goType)
	l.Add("const (")
	l.Indent++
	for _, value := range ds.Enum {
		enumValueName := ""
		if ds.Type == "string" {
			enumValueName = ToTitle(value.(string))
		} else {
			enumValueName = fmt.Sprintf("%v", value)
		}
		valueTitle := enumTypeName + enumValueName
		// MyEnumName EnumType = EnumValue
		if ds.Type == "string" {
			l.Add("%s %s = \"%s\"", valueTitle, enumTypeName, value)
		} else {
			l.Add("%s %s = %v", valueTitle, enumTypeName, value)
		}
	}
	l.Indent--
	l.Add(")")
	l.Add("")
}

// GenDataSchemaOneOf generates enum constants from 'onOff' schema definition
func GenDataSchemaOneOf(l *SL, enumTypeName string, ds *td.DataSchema) (err error) {
	goType := GoTypeFromSchema(ds)
	l.Add("// %s enumerator", enumTypeName)
	GenDescription(l, ds.Description, ds.Comments)
	l.Add("type %s %s", enumTypeName, goType)
	l.Add("const (")
	l.Indent++
	for _, dsEnum := range ds.OneOf {
		enumValue := dsEnum.Const
		if enumValue == nil {
			err = fmt.Errorf("missing oneOf value in 'const' field")
			l.Add("// Error: Misssing oneOf value in 'const' field")
			continue
		}
		enumValueName := ""
		// The enum name is the typename followed by the value in title case
		if ds.Type == "string" {
			enumValueName = ToTitle(enumValue.(string))
		} else {
			enumValueName = fmt.Sprintf("%v", enumValue)
		}
		valueTitle := enumTypeName + enumValueName
		l.Add("")
		l.Add("// %s for %s", valueTitle, dsEnum.Title)
		GenDescription(l, dsEnum.Description, dsEnum.Comments)
		//if ds.Type == "string" {
		//	l.Add("%s = \"%s\"", valueTitle, enumValue)
		//} else {
		l.Add("%s %s = \"%v\"", valueTitle, enumTypeName, enumValue)
		//}
		//GenSchemaDefConst(l, enumValueName, &dsEnum)
	}

	l.Indent--
	l.Add(")")
	l.Add("")
	return err
}
