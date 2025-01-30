package gentypes

import (
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/td"
	"golang.org/x/exp/slices"
)

func GenDataSchema(l *utils.SL, agentID, schemaName string, ds *td.DataSchema) (err error) {
	schemaTypeName := ToTitle(schemaName)

	if ds.Type == "object" {
		// generate a complex type struct
		if ds.Ref == "" {
			// define an agent scoped data struct
			err = GenDataSchemaStruct(l, agentID, schemaTypeName, ds)
		} else {
			// $ref links to an existing schema. Nothing to do here.
		}
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

// GenDataSchemaStruct generates a data struct scoped to the agent from schema definition
func GenDataSchemaStruct(l *utils.SL, agentID string, idTitle string, ds *td.DataSchema) error {
	l.Add("// %s defines a %s data schema of the %s agent.", idTitle, ds.Title, agentID)
	GenDescription(l, ds.Description, ds.Comments)
	l.Add("type %s struct {", idTitle)

	l.Indent++
	// define an agent wide data struct
	err := GenDataSchemaFields(l, idTitle, ds)
	l.Indent--

	l.Add("}")
	l.Add("")
	return err
}

// GenDataSchemaFields generates the fields of a golang struct from its DataSchema.
// Intended for generating fields in action, event, property affordances and schema definitions.
//
//	l is the output lines with generated source code
//	name is the field name of the dataschema
//	ds is the dataschema to generate
func GenDataSchemaFields(l *utils.SL, name string, ds *td.DataSchema) (err error) {
	// get the list of attributes in this schema
	//attrList := GetSchemaAttrs(key, ds, true)
	// the top level attribute can be a single attribute or a list of properties
	//schemaAttr := GetSchemaAttr(key, ds)
	//if schemaAttr.Nested != nil {
	if len(ds.Properties) > 0 {
		// field is a dataschema
		err = GenSchemaAttr(l, ds.Properties)
	} else {
		props := map[string]*td.DataSchema{name: ds}
		err = GenSchemaAttr(l, props)
	}
	return err
}

// GenDescription generates the comments from description and comments
func GenDescription(l *utils.SL, description string, comments []string) {
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
// func GenSchemaAttr(l *utils.SL, attrList []SchemaAttr) {
func GenSchemaAttr(l *utils.SL, attrMap map[string]*td.DataSchema) (err error) {

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
		} else if attr.Ref != "" {
			// field is a reference to a dataschema
			typeName := ToTitle(attr.Ref)
			l.Add("%s %s", keyTitle, typeName)
		} else if len(attr.AdditionalProperties) > 0 {
			// field is a map of dataschema
			//GenSchemaAttr(l, attr.AdditionalProperties)
		} else {
			omitEmpty := ""
			isRequired := false
			if attr.Required != nil {
				isRequired = slices.Contains(attr.Required, key)
			}

			if !isRequired {
				omitEmpty = ",omitempty"
			}
			l.Add("%s %s `json:\"%s%s\"`", keyTitle, goType, key, omitEmpty)
		}
	}
	return err
}

// GenDataSchemaConst generates a constant value from schema definition
func GenDataSchemaConst(l *utils.SL, title string, ds *td.DataSchema) {
	l.Add("")
	l.Add("// %s constant with %s", title, ds.Description)
	l.Add("const %s = %s", title, ds.Const)
}

// GenDataSchemaEnum generates enum type and constants from schema definition
func GenDataSchemaEnum(l *utils.SL, enumTypeName string, ds *td.DataSchema) {

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
func GenDataSchemaOneOf(l *utils.SL, enumTypeName string, ds *td.DataSchema) (err error) {
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
