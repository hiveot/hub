package tdd2go

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
)

// GenSchemaDefinitions generates data structs and constants defined in schema definitions
func GenSchemaDefinitions(l *utils.L, serviceTitle string, td *things.TD) {

	agentID, _ := things.SplitDigiTwinThingID(td.ID)

	if len(td.SchemaDefinitions) > 0 {
		l.Add("")
		l.Add("//--- Schema definitions of Thing '%s' ---", td.ID)
		l.Add("")
	}

	keys := utils.OrderedMapKeys(td.SchemaDefinitions)

	for _, id := range keys {
		sd := td.SchemaDefinitions[id]
		idTitle := ToTitle(id)

		if sd.Type == "object" {
			if sd.Ref == "" {
				// define an agent wide data struct
				GenSchemaDefStruct(l, agentID, idTitle, sd)
			} else {
				// nothing to do here
			}
		} else if sd.Enum != nil {
			GenSchemaDefEnum(l, idTitle, sd)
		} else if sd.OneOf != nil {
			GenSchemaDefOneOf(l, idTitle, sd)
		} else if sd.Default != "" {
			// define an agent wide constant
			GenSchemaDefConst(l, idTitle, sd)
		} else if sd.Const != nil {
			GenSchemaDefConst(l, idTitle, sd)
		}
	}
}

// GenSchemaDefStruct generates a data struct scoped to the agent from schema definition
func GenSchemaDefStruct(l *utils.L, agentID string, idTitle string, ds *things.DataSchema) {
	l.Add("// %s defines a %s data schema of the %s agent.", idTitle, ds.Title, agentID)
	GenDescription(l, ds.Description, ds.Comments)
	l.Add("type %s struct {", idTitle)

	l.Indent++
	// define an agent wide data struct
	GenDataSchemaFields(l, idTitle, ds)
	l.Indent--

	l.Add("}")
	l.Add("")
}

// GenSchemaDefConst generates a constant value from schema definition
func GenSchemaDefConst(l *utils.L, title string, ds *things.DataSchema) {
	l.Add("")
	l.Add("// %s constant with %s", title, ds.Description)
	l.Add("const %s = %s", title, ds.Const)
}

// GenSchemaDefEnum generates enum type and constants from schema definition
func GenSchemaDefEnum(l *utils.L, enumTypeName string, ds *things.DataSchema) {

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

// GenSchemaDefOneOf generates enum constants from 'onOff' schema definition
func GenSchemaDefOneOf(l *utils.L, enumTypeName string, ds *things.DataSchema) {
	goType := GoTypeFromSchema(ds)
	l.Add("// %s enumerator", enumTypeName)
	GenDescription(l, ds.Description, ds.Comments)
	l.Add("type %s %s", enumTypeName, goType)
	l.Add("const (")
	l.Indent++
	for _, dsEnum := range ds.OneOf {
		enumValue := dsEnum.Const
		if enumValue == nil {
			slog.Error("Missing oneOf value in 'const' field")
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
		if ds.Type == "string" {
			l.Add("%s = \"%s\"", valueTitle, enumValue)
		} else {
			l.Add("%s %s = %v", valueTitle, enumTypeName, enumValue)
		}
		//GenSchemaDefConst(l, enumValueName, &dsEnum)
	}

	l.Indent--
	l.Add(")")
	l.Add("")
}

// GenDescription generates the comments from description and comments
func GenDescription(l *utils.L, description string, comments []string) {
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
