package tdd2go

import (
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"golang.org/x/exp/slices"
)

// GenDataSchemaFields generates the fields of a golang struct from its DataSchema.
// Intended for generating fields in action, event, property affordances and schema definitions.
//
//	l is the output lines with generated source code
//	key is the field name of the dataschema
//	ds is the dataschema to generate
func GenDataSchemaFields(l *utils.L, key string, ds *things.DataSchema) {
	// get the list of attributes in this schema
	//attrList := GetSchemaAttrs(key, ds, true)
	// the top level attribute can be a single attribute or a list of properties
	//schemaAttr := GetSchemaAttr(key, ds)
	//if schemaAttr.Nested != nil {
	if len(ds.Properties) > 0 {
		//GenSchemaAttr(l, schemaAttr.Nested)
		GenSchemaAttr(l, ds.Properties)
	} else {
		props := map[string]*things.DataSchema{key: ds}
		GenSchemaAttr(l, props)
	}
}

// GenSchemaAttr generates the attribute fields of a dataschema
//
//	attrMap contains a map of attribute keys with their description
//
// func GenSchemaAttr(l *utils.L, attrList []SchemaAttr) {
func GenSchemaAttr(l *utils.L, attrMap map[string]*things.DataSchema) {

	keys := utils.OrderedMapKeys(attrMap)
	for _, key := range keys {
		attr := attrMap[key]
		keyTitle := ToTitle(key)
		goType := GoTypeFromSchema(attr)
		l.Add("")
		l.Add("// %s with %s", keyTitle, attr.Title)
		GenDescription(l, attr.Description, attr.Comments)
		if attr.Properties != nil {
			// nested struct
			GenSchemaAttr(l, attr.Properties)
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
}
