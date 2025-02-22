package directorycli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/wot/td"
	"github.com/urfave/cli/v2"
	"log/slog"
)

func DirectoryListCommand(co **messaging.Consumer) *cli.Command {
	var verbose = false
	return &cli.Command{
		Name:      "ld",
		Category:  "directory",
		Usage:     "List directory of Things",
		ArgsUsage: "[<thingID>]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "raw",
				Usage:       "raw, display raw json",
				Value:       false,
				Destination: &verbose,
			},
		},
		Action: func(cCtx *cli.Context) error {
			var err = fmt.Errorf("expected 0 or 1 parameters")
			if cCtx.NArg() == 0 {
				err = HandleListDirectory(*co)
			} else if cCtx.NArg() == 1 {
				if !verbose {
					err = HandleListThing(*co, cCtx.Args().First())
				} else {
					err = HandleListThingVerbose(*co, cCtx.Args().First())
				}
			}
			return err
		},
	}
}

// HandleListDirectory lists the directory content
func HandleListDirectory(co *messaging.Consumer) (err error) {
	// todo: iterate with offset and limit
	tdListJson, err := digitwin.ThingDirectoryReadAllTDs(co, 300, 0)
	tdList, err2 := td.UnmarshalTDList(tdListJson)

	if err != nil || err2 != nil {
		return err
	}
	fmt.Printf("Thing ID                            @type                               Title                                #props  #events #actions   Modified         \n")
	fmt.Printf("----------------------------------  ----------------------------------  -----------------------------------  ------  ------- --------   -----------------------------\n")
	for _, tdDoc := range tdList {
		var updatedStr = ""
		if tdDoc.Modified != "" {
			updatedStr = tputils.DecodeAsDatetime(tdDoc.Modified)
		} else if tdDoc.Created != "" {
			updatedStr = tputils.DecodeAsDatetime(tdDoc.Created)
		}

		fmt.Printf("%-35s %-35.35s %-35.35s %7d  %7d  %7d   %-30s\n",
			tdDoc.ID,
			tdDoc.AtType,
			tdDoc.Title,
			len(tdDoc.Properties),
			len(tdDoc.Events),
			len(tdDoc.Actions),
			updatedStr,
		)
	}
	fmt.Println()
	return nil
}

// HandleListThing lists details of a Thing in the directory
func HandleListThing(co *messaging.Consumer, thingID string) error {

	tdDocJson, err := digitwin.ThingDirectoryReadTD(co, thingID)
	tdDoc, err2 := td.UnmarshalTD(tdDocJson)
	if err != nil || err2 != nil {
		return err
	}
	propValueMap, err := digitwin.ThingValuesReadAllProperties(co, thingID)

	if err != nil {
		slog.Error("Unable to read directory:", "err", err)
	}
	fmt.Printf("%sTD of %s    %s\n", utils.COBlue, thingID, utils.COReset)
	fmt.Printf(" title:       %s\n", tdDoc.Title)
	fmt.Printf(" description: %s\n", tdDoc.Description)
	fmt.Printf(" @type:       %s\n", tdDoc.AtType)
	fmt.Printf(" modified:    %s\n", tputils.DecodeAsDatetime(tdDoc.Modified))
	fmt.Println("")

	fmt.Println(utils.COGreen + "Attributes:")
	fmt.Println(" ID                             Title                                    Value           Description")
	fmt.Println(" ----------------------------   ---------------------------------------  -------------   -----------" + utils.COReset)
	keys := utils.OrderedMapKeys(tdDoc.Properties)
	for _, key := range keys {
		prop, found := tdDoc.Properties[key]
		if found && prop.ReadOnly {
			value := propValueMap[key]
			valueStr := tputils.DecodeAsString(value.Output, 15)
			fmt.Printf(" %-30s %-40.40s %s%-15.15s%s %-.80s\n",
				key, prop.Title, utils.COGreen, valueStr, utils.COReset, prop.Description)
		}
	}
	fmt.Println()
	fmt.Println(utils.COBlue + "Configuration:")
	fmt.Println(" ID                             Title                                    DataType   Value                Description")
	fmt.Println(" -----------------------------  ---------------------------------------  ---------  ------------------   -----------" + utils.COReset)
	for _, key := range keys {
		prop, found := tdDoc.Properties[key]
		if found && !prop.ReadOnly {
			value := propValueMap[key]
			valueStr := tputils.DecodeAsString(value.Output, 15)
			fmt.Printf(" %-30s %-40.40s %-10.10s %s%-15.15s%s %-.80s\n",
				key, prop.Title, prop.Type, utils.COBlue, valueStr, utils.COReset, prop.Description)
		}
	}

	fmt.Println(utils.COYellow + "\nEvents:")
	fmt.Println(" ID                                  EventType                 Title                                    DataType   Value           Description")
	fmt.Println(" ----------------------------------  ------------------------  ---------------------------------------  ---------  --------------  -----------" + utils.COReset)
	eventValueMap, err := digitwin.ThingValuesReadAllEvents(co, thingID)
	keys = utils.OrderedMapKeys(tdDoc.Events)
	for _, key := range keys {
		ev := tdDoc.Events[key]
		dataType := "(n/a)"
		if ev.Data.Type != "" {
			dataType = ev.Data.Type
		}
		value := eventValueMap[key]
		valueStr := tputils.DecodeAsString(value.Output, 15)
		if ev.Data.Type != "" {
			//initialValue = ev.Data.InitialValue
		}
		fmt.Printf(" %-35s %-25.25s %-40.40s %-10.10v %s%-15.15s%s %.80s\n",
			key, ev.GetAtTypeString(), ev.Title, dataType, utils.COYellow, valueStr, utils.COReset, ev.Description)
	}

	fmt.Println(utils.CORed + "\nActions:")
	fmt.Println(" ID                             ActionType                Title                                    Arg(s)     Value           Description")
	fmt.Println(" -----------------------------  ------------------------  ---------------------------------------  ---------  --------------  -----------" + utils.COReset)
	actionValueMap, err := digitwin.ThingValuesReadAllProperties(co, thingID)
	keys = utils.OrderedMapKeys(tdDoc.Actions)
	for _, key := range keys {
		action := tdDoc.Actions[key]
		dataType := "(n/a)"
		value := actionValueMap[key]
		valueStr := tputils.DecodeAsString(value.Output, 15)
		if action.Input != nil {
			dataType = action.Input.Type
			//initialValue = action.Input.InitialValue
		}
		fmt.Printf(" %-30.30s %-25.25s %-40.40s %-10.10s %s%-15.15s%s %.80s\n",
			key, action.AtType, action.Title, dataType, utils.CORed, valueStr, utils.COReset, action.Description)
	}
	fmt.Println()
	return err
}

// HandleListThingVerbose lists a Thing full TD
func HandleListThingVerbose(co *messaging.Consumer, thingID string) error {
	tdJSON, err := digitwin.ThingDirectoryReadTD(co, thingID)

	if err != nil {
		return err
	}
	fmt.Println("TD of", thingID)
	var buf bytes.Buffer
	_ = json.Indent(&buf, []byte(tdJSON), "", "\t")
	fmt.Printf("%s\n", buf.Bytes())
	return err
}
