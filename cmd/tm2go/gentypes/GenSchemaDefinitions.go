package gentypes

import (
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/td"
)

// GenSchemaDefinitions generates golang structs from the dataschema in the SchemaDefinitions section
func GenSchemaDefinitions(l *utils.SL, td1 *td.TD) error {

	//agentID, _ := td.SplitDigiTwinThingID(td1.ID)

	if len(td1.SchemaDefinitions) > 0 {
		l.Add("")
		l.Add("//--- Schema definitions of Thing '%s' ---", td1.ID)
		l.Add("")
	}

	keys := utils.OrderedMapKeys(td1.SchemaDefinitions)

	for _, schemaName := range keys {
		dataSchema := td1.SchemaDefinitions[schemaName]
		schemaTypeName := ToTitle(schemaName)

		err := GenDataSchema(l, schemaTypeName, &dataSchema)
		if err != nil {
			l.Add("// Aborted due to error: %s", err.Error())
			l.Add("")
			return err
		}
	}
	return nil
}
