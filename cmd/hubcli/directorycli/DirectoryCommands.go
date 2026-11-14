package directorycli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/cells/directory"
	consumer_recipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/urfave/cli/v3"
)

func DirectoryListCommand(r **consumer_recipe.ConsumerRecipe) *cli.Command {
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
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dirClient := (*r).GetDirectory()
			co := (*r).Consumer
			var err = fmt.Errorf("expected 0 or 1 parameters")
			if cmd.NArg() == 0 {
				err = HandleListDirectory(dirClient)
			} else if cmd.NArg() == 1 {
				if !verbose {
					err = HandleListThing(dirClient, co, cmd.Args().First())
				} else {
					err = HandleListThingVerbose(dirClient, co, cmd.Args().First())
				}
			}
			return err
		},
	}
}

// HandleListDirectory lists the directory content
func HandleListDirectory(dirClient directory.IDirectoryClient) (err error) {
	// todo: iterate with offset and limit

	tdList, cacheOnly := dirClient.RetrieveAllThings(0, 300)
	wot.SortThingsByID(tdList)
	if cacheOnly {
		fmt.Printf("Directory server not reachable. Using cache only\n")
	}

	fmt.Printf("Thing ID                            Title                                 @type                                #props  #events #actions   Modified         \n")
	fmt.Printf("----------------------------------  ------------------------------------  -----------------------------------  ------  ------- --------   -----------------------------\n")
	for _, tdDoc := range tdList {
		var updatedStr = ""
		if tdDoc.Modified != "" {
			//updatedStr = utils.DecodeAsDatetime(tdDoc.Modified)
			updatedStr = utils.FormatDateTime(tdDoc.Modified)
		} else if tdDoc.Created != "" {
			//updatedStr = utils.DecodeAsDatetime(tdDoc.Created)
			updatedStr = utils.FormatDateTime(tdDoc.Created)
		}

		fmt.Printf("%-35s %-37.37s %-35.35s %7d  %7d  %7d   %-30s\n",
			tdDoc.ID,
			tdDoc.Title,
			tdDoc.AtType,
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
func HandleListThing(dirClient directory.IDirectoryClient, co *consumer.Consumer, thingID string) error {

	tdDoc, err := dirClient.RetrieveThing(thingID)
	if err != nil {
		return err
	}
	propValueMap, err := co.ReadAllProperties(thingID)

	if err != nil {
		slog.Error("Unable to read directory:", "err", err)
	}
	fmt.Printf("%sTD of %s    %s\n", utils.COBlue, thingID, utils.COReset)
	fmt.Printf(" title:       %s\n", tdDoc.Title)
	fmt.Printf(" description: %s\n", tdDoc.Description)
	fmt.Printf(" @type:       %s\n", tdDoc.AtType)
	fmt.Printf(" modified:    %s\n", utils.FormatDateTime(tdDoc.Modified))
	fmt.Println("")

	fmt.Println(utils.COGreen + "Attributes:")
	fmt.Println(" ID                             Title                                    Value           Description")
	fmt.Println(" ----------------------------   ---------------------------------------  -------------   -----------" + utils.COReset)
	keys := utils.OrderedMapKeys(tdDoc.Properties)
	for _, key := range keys {
		prop, found := tdDoc.Properties[key]
		if found && prop.ReadOnly {
			propValue := propValueMap[key]
			valueStr := utils.DecodeAsString(propValue, 15)
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
			propValue := propValueMap[key]
			valueStr := utils.DecodeAsString(propValue, 15)
			fmt.Printf(" %-30s %-40.40s %-10.10s %s%-15.15s%s %-.80s\n",
				key, prop.Title, prop.Type, utils.COBlue, valueStr, utils.COReset, prop.Description)
		}
	}

	fmt.Println(utils.COYellow + "\nEvents:")
	fmt.Println(" ID                                  EventType                 Title                                    DataType   Value           Description")
	fmt.Println(" ----------------------------------  ------------------------  ---------------------------------------  ---------  --------------  -----------" + utils.COReset)
	eventValueMap, err := co.ReadAllEvents(thingID)
	keys = utils.OrderedMapKeys(tdDoc.Events)
	for _, key := range keys {
		ev := tdDoc.Events[key]
		dataType := "(n/a)"
		if ev.Data.Type != "" {
			dataType = ev.Data.Type
		}
		value := eventValueMap[key]
		valueStr := value.ToString(15)
		if ev.Data.Type != "" {
			//initialValue = ev.Data.InitialValue
		}
		fmt.Printf(" %-35s %-25.25s %-40.40s %-10.10v %s%-15.15s%s %.80s\n",
			key, ev.GetAtTypeString(), ev.Title, dataType, utils.COYellow, valueStr, utils.COReset, ev.Description)
	}

	fmt.Println(utils.CORed + "\nActions:")
	fmt.Println(" ID                                  ActionType                Title                                    Arg(s)     Value           Description")
	fmt.Println(" ----------------------------------  ------------------------  ---------------------------------------  ---------  --------------  -----------" + utils.COReset)
	actionValueMap, err := co.QueryAllActions(thingID)
	keys = utils.OrderedMapKeys(tdDoc.Actions)
	for _, key := range keys {
		action := tdDoc.Actions[key]
		dataType := "(n/a)"
		actionResp := actionValueMap[key]
		valueStr := utils.DecodeAsString(actionResp.Output, 15)
		if action.Input != nil {
			dataType = action.Input.Type
			//initialValue = action.Input.InitialValue
		}
		fmt.Printf(" %-30.30s %-25.25s %-40.40s %-10.10s %s%-15.15s%s %.80s\n",
			key, action.GetAtTypeString(), action.Title, dataType, utils.CORed, valueStr, utils.COReset, action.Description)
	}
	fmt.Println()
	return err
}

// HandleListThingVerbose lists a Thing full TD
func HandleListThingVerbose(dirClient directory.IDirectoryClient, co *consumer.Consumer, thingID string) error {
	tdoc, err := dirClient.RetrieveThing(thingID)

	if err != nil {
		return err
	}
	fmt.Println("TD of", thingID)
	var buf bytes.Buffer
	tdJSON := wot.MarshalTD(tdoc)
	_ = json.Indent(&buf, []byte(tdJSON), "", "\t")
	fmt.Printf("%s\n", buf.Bytes())
	return err
}
