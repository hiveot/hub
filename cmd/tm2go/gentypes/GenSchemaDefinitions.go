package gentypes

import (
	"github.com/hiveot/hivekitgo/utils"
	"github.com/hiveot/hivekitgo/wot/td"
)

// GenSchemaDefinitions generates golang structs from the dataschema in the SchemaDefinitions section
func GenSchemaDefinitions(l *SL, td1 *td.TD) error {

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
