package _go

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
)

type SchemaAttr struct {
	// Key of the dataschema
	Key string
	// Name of the attribute
	AttrName string
	// Golang type of attribute
	AttrType string
	// Description
	Description string
}

// GetSchemaAttrs returns a list of attributes defined in the DataSchema
//
// FIXME: The order of multiple arguments is the order in which the properties map is iterated.
// The map order seems to be that of the order in which the TDD has it defined but this
// is still a question.
//
// If the data schema is a native type, then return the single attribute
// If the data schema is an object, then return a list of attributes, one for each member
// Schema vocabulary is converted to golang
func GetSchemaAttrs(key string, schema *things.DataSchema) []SchemaAttr {
	attrList := make([]SchemaAttr, 0, 5)
	if schema == nil {
		return attrList
	}

	if schema.Type == vocab.WoTDataTypeObject {
		// schema is an ObjectSchema with properties
		// add an attribute for each member
		if schema.Properties != nil {
			for key, value := range schema.Properties {
				attrList = append(attrList, GetSchemaAttrs(key, value)...)
			}
		} else {
			// an object schema without properties is unexpected
			attrList = append(attrList, SchemaAttr{
				Key:         key,
				AttrName:    Key2ID(key),
				AttrType:    "string",
				Description: fmt.Sprintf("Object Schema '%s' without attributes is unexpected", schema.Title),
			})
		}
	} else if schema.Type == vocab.WoTDataTypeArray {
		goType := GetSchemaType(schema)
		description := schema.Title
		if schema.Description == "" {
			description = schema.Description
		}
		attrList = append(attrList, SchemaAttr{
			Key:         key,
			AttrName:    Key2ID(key),
			AttrType:    goType,
			Description: description,
		})
	} else {
		// native type
		goType := GetSchemaType(schema)
		description := schema.Title
		if schema.Description == "" {
			description = schema.Description
		}
		attrList = append(attrList, SchemaAttr{
			Key:         key,
			AttrName:    Key2ID(key),
			AttrType:    goType,
			Description: description,
		})
	}
	return attrList
}

// GenDataSchemaParams generate a golang variable or list of variables from a dataschema
func GenDataSchemaParams(l *utils.L, attrList []SchemaAttr) {
	for _, attr := range attrList {
		l.Add("")
		l.Add("    // %s %s", attr.AttrName, attr.Description)
		l.Add("    %s %s `json:\"%s\"`", attr.AttrName, attr.AttrType, attr.AttrName)

	}
}
