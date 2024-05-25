package _go

import (
	"github.com/hiveot/hub/lib/utils"
)

// GenDataSchemaParams generate a golang variable or list of variables from a dataschema
func GenDataSchemaParams(l *utils.L, attrList []SchemaAttr) {
	for _, attr := range attrList {
		l.Add("")
		l.Add("// %s %s", attr.AttrName, attr.Description)
		l.Add("%s %s `json:\"%s\"`", attr.AttrName, attr.AttrType, attr.Key)

	}
}
