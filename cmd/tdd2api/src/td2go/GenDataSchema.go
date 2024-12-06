package td2go

import (
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/td"
	"golang.org/x/exp/slices"
)

// GenDataSchemaFields generates the fields of a golang struct from its DataSchema.
// Intended for generating fields in action, event, property affordances and schema definitions.
//
//	l is the output lines with generated source code
//	name is the field name of the dataschema
//	ds is the dataschema to generate
func GenDataSchemaFields(l *utils.SL, name string, ds *td.DataSchema) {
	// get the list of attributes in this schema
	//attrList := GetSchemaAttrs(key, ds, true)
	// the top level attribute can be a single attribute or a list of properties
	//schemaAttr := GetSchemaAttr(key, ds)
	//if schemaAttr.Nested != nil {
	if len(ds.Properties) > 0 {
		// field is a dataschema
		GenSchemaAttr(l, ds.Properties)
	} else {
		props := map[string]*td.DataSchema{name: ds}
		GenSchemaAttr(l, props)
	}
}

// GenSchemaAttr generates the attribute fields of a dataschema
//
//	attrMap contains a map of attribute name with their description
//
// func GenSchemaAttr(l *utils.SL, attrList []SchemaAttr) {
func GenSchemaAttr(l *utils.SL, attrMap map[string]*td.DataSchema) {

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
			GenSchemaAttr(l, attr.Properties)
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
}
