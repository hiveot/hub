package _go

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"golang.org/x/exp/slices"
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
	// Required attribute
	Required bool

	// nested struct
	Nested []SchemaAttr
}

// GenDataSchema generates a golang data struct from a DataSchema definition
//
//	l is the output lines with generated source code
//	key is the field name of the dataschema
//	ds is the dataschema to generate
func GenDataSchema(l *utils.L, key string, ds *things.DataSchema) {
	// get the list of attributes in this schema
	//attrList := GetSchemaAttrs(key, ds, true)
	// the top level attribute can be a single attribute or a list of properties
	schemaAttr := GetSchemaAttr(key, ds)
	if schemaAttr.Nested != nil {
		GenSchemaAttr(l, schemaAttr.Nested)
	} else {
		GenSchemaAttr(l, []SchemaAttr{schemaAttr})
	}
}

func GenSchemaAttr(l *utils.L, attrList []SchemaAttr) {
	for _, attr := range attrList {
		l.Add("")
		l.Add("// %s %s", attr.AttrName, attr.Description)
		if attr.Nested != nil {
			// nested struct
			GenSchemaAttr(l, attr.Nested)
		} else {
			omitEmpty := ""
			if !attr.Required {
				omitEmpty = ",omitempty"
			}
			l.Add("%s %s `json:\"%s%s\"`", attr.AttrName, attr.AttrType, attr.Key, omitEmpty)
		}
	}
}

// GetSchemaAttrs returns a list of attributes defined in the DataSchema
// FIXME: This does not support nested schemas
//
// If the data schema is a native type, then return the single attribute
// If the data schema is an object, then return a list of attributes, one for each member
// Schema vocabulary is converted to golang
//func GetSchemaAttrs(key string, schema *things.DataSchema, topLevel bool) []SchemaAttr {
//	attrList := make([]SchemaAttr, 0, 5)
//	if schema == nil {
//		return attrList
//	}
//	isRequired := false
//	if schema.Required != nil {
//		slog.Warn("KEY is required", "key", key)
//		isRequired = slices.Contains(schema.Required, key)
//	}
//	goType := GetSchemaType(schema)
//	description := schema.Title
//	if schema.Description != "" {
//		description = schema.Description
//	}
//	if schema.Type == vocab.WoTDataTypeObject {
//		// schema is an ObjectSchema with properties
//		// add a list for each property
//		if schema.Properties == nil {
//			schemaAttr := SchemaAttr{
//				Key:         key,
//				AttrName:    Key2ID(key),
//				AttrType:    goType,
//				Description: description,
//				Required:    isRequired,
//			}
//			attrList = append(attrList, schemaAttr)
//		} else {
//			// if the object has properties then add the list of properties instead
//			nested := make([]SchemaAttr, 0, len(schema.Properties))
//			for key, value := range schema.Properties {
//				sa := GetSchemaAttrs(key, value, false)
//				if schema.Required != nil {
//					slog.Warn("KEY is required", "key", key)
//					//sa.Required = slices.Contains(schema.Required, key)
//				}
//				nested = append(nested, sa...)
//			}
//			attrList = append(attrList, nested...)
//			//schemaAttr.Nested = nested
//		}
//	} else if schema.Type == vocab.WoTDataTypeArray {
//		attrList = append(attrList, SchemaAttr{
//			Key:         key,
//			AttrName:    Key2ID(key),
//			AttrType:    goType,
//			Description: description,
//			Required:    isRequired,
//		})
//	} else {
//		// native type
//		attrList = append(attrList, SchemaAttr{
//			Key:         key,
//			AttrName:    Key2ID(key),
//			AttrType:    goType,
//			Description: description,
//			Required:    isRequired,
//		})
//	}
//	return attrList
//}

// GetSchemaAttr returns an attribute definition in the DataSchema
//
// key is the property ID in the schema 'properties' map
func GetSchemaAttr(key string, schema *things.DataSchema) (schemaAttr SchemaAttr) {
	//isRequired := false
	//if schema.Required != nil {
	//	slog.Warn("KEY is required", "key", key)
	//	isRequired = slices.Contains(schema.Required, key)
	//}
	goType := GetSchemaType(schema)
	description := schema.Title
	if schema.Description != "" {
		description = schema.Description
	}
	if schema.Type == vocab.WoTDataTypeObject {
		// attribute is an ObjectSchema with properties
		// add a nested list for each property
		schemaAttr = SchemaAttr{
			Key:         key,
			AttrName:    Key2ID(key),
			AttrType:    goType,
			Description: description,
			Required:    true,
		}
		// this property has nested properties
		if schema.Properties != nil {
			// if the object has properties then add the list of properties instead
			nested := make([]SchemaAttr, 0, len(schema.Properties))
			for key, value := range schema.Properties {
				sa2 := GetSchemaAttr(key, value)
				// An object dataschema has an array of required property names
				if schema.Required != nil {
					sa2.Required = slices.Contains(schema.Required, key)
				} else {
					sa2.Required = false
				}
				nested = append(nested, sa2)
			}
			schemaAttr.Nested = nested
		}
	} else if schema.Type == vocab.WoTDataTypeArray {
		schemaAttr = SchemaAttr{
			Key:         key,
			AttrName:    Key2ID(key),
			AttrType:    goType,
			Description: description,
			Required:    true,
		}
	} else {
		// native type
		schemaAttr = SchemaAttr{
			Key:         key,
			AttrName:    Key2ID(key),
			AttrType:    goType,
			Description: description,
			Required:    true,
		}
	}
	return schemaAttr
}
