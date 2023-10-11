package directorycli

import (
	"encoding/json"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/core/directory"
	"github.com/hiveot/hub/core/directory/service"
	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/lib/utils"
	"github.com/urfave/cli/v2"
	"time"

	"github.com/hiveot/hub/lib/hubclient"
)

func DirectoryListCommand(hc *hubclient.IHubClient) *cli.Command {
	var verbose = false
	return &cli.Command{
		Name:      "ld",
		Category:  "directory",
		Usage:     "List directory of Things",
		ArgsUsage: "[<agentID> <thingID>]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "v",
				Usage:       "Verbose, display raw json",
				Value:       false,
				Destination: &verbose,
			},
		},
		Action: func(cCtx *cli.Context) error {
			var err = fmt.Errorf("expected 0 or 2 parameters")
			if cCtx.NArg() == 0 {
				err = HandleListDirectory(*hc)
			} else if cCtx.NArg() == 2 {
				if !verbose {
					err = HandleListThing(*hc, cCtx.Args().First(), cCtx.Args().Get(1))
				} else {
					err = HandleListThingVerbose(*hc, cCtx.Args().First(), cCtx.Args().Get(1))
				}
			}
			return err
		},
	}
}

// HandleListDirectory lists the directory content
func HandleListDirectory(hc hubclient.IHubClient) (err error) {
	offset := 0
	limit := 100
	rdir := service.NewReadDirectoryService(hc)

	cursor := rdir.GetReadCursor()
	fmt.Printf("Agent ID        Thing ID             Device Type          Title                                #props  #events #actions   Modified         \n")
	fmt.Printf("-------------   -------------------  -------------------  -----------------------------------  ------  ------- --------   --------------------------\n")
	i := 0
	tv, valid := cursor.First()
	if offset > 0 {
		// TODO, skip
		//tv, valid = cursor.Skip(offset)
	}
	for ; valid && i < limit; tv, valid = cursor.Next() {
		var tdDoc thing.TD
		err = json.Unmarshal(tv.Data, &tdDoc)
		var utime time.Time
		if tdDoc.Modified != "" {
			utime, err = dateparse.ParseAny(tdDoc.Modified)
		} else if tdDoc.Created != "" {
			utime, err = dateparse.ParseAny(tdDoc.Created)
		}
		timeStr := utime.In(time.Local).Format("02 Jan 2006 15:04:05 -0700")

		fmt.Printf("%-15s %-20s %-20.20s %-35.35s %7d  %7d  %7d   %-30s\n",
			tv.AgentID,
			tdDoc.ID,
			tdDoc.DeviceType,
			tdDoc.Title,
			len(tdDoc.Properties),
			len(tdDoc.Events),
			len(tdDoc.Actions),
			timeStr,
		)
	}
	fmt.Println()
	return nil
}

// HandleListThing lists details of a Thing in the directory
func HandleListThing(hc hubclient.IHubClient, pubID, thingID string) error {
	var tdDoc thing.TD

	rdir := service.NewReadDirectoryService(hc)
	tv, err := rdir.GetTD(pubID, thingID)
	if err != nil {
		return err
	}
	err = json.Unmarshal(tv.Data, &tdDoc)
	if err != nil {
		return err
	}
	fmt.Printf("%sTD of %s %s:%s\n", utils.COBlue, pubID, thingID, utils.COReset)
	fmt.Printf(" title:       %s\n", tdDoc.Title)
	fmt.Printf(" description: %s\n", tdDoc.Description)
	fmt.Printf(" deviceType:  %s\n", tdDoc.DeviceType)
	fmt.Printf(" modified:    %s\n", tdDoc.Modified)
	fmt.Println("")

	fmt.Println(utils.COGreen + "Attributes:")
	fmt.Println(" ID                             Title                                    Initial Value   Description")
	fmt.Println(" ----------------------------   ---------------------------------------  -------------   -----------" + utils.COReset)
	keys := utils.OrderedMapKeys(tdDoc.Properties)
	for _, key := range keys {
		prop, found := tdDoc.Properties[key]
		if found && prop.ReadOnly {
			fmt.Printf(" %-30s %-40.40s %s%-15.15v%s %s\n",
				key, prop.Title, utils.COGreen, prop.InitialValue, utils.COReset, prop.Description)
		}
	}
	fmt.Println()
	fmt.Println(utils.COBlue + "Configuration:")
	fmt.Println(" ID                             Title                                    DataType   Initial Value   Description")
	fmt.Println(" -----------------------------  ---------------------------------------  ---------  -------------   -----------" + utils.COReset)
	for _, key := range keys {
		prop, found := tdDoc.Properties[key]
		if found && !prop.ReadOnly {
			fmt.Printf(" %-30s %-40.40s %-10s %s%-15.15v%s %s\n",
				key, prop.Title, prop.Type, utils.COBlue, prop.InitialValue, utils.COReset, prop.Description)
		}
	}

	fmt.Println(utils.COYellow + "\nEvents:")
	fmt.Println(" ID                             EventType       Title                                    DataType   Initial Value   Description")
	fmt.Println(" -----------------------------  --------------  ---------------------------------------  ---------  --------------  -----------" + utils.COReset)
	keys = utils.OrderedMapKeys(tdDoc.Events)
	for _, key := range keys {
		ev := tdDoc.Events[key]
		dataType := "(n/a)"
		if ev.Data != nil {
			dataType = ev.Data.Type
		}
		initialValue := ""
		if ev.Data != nil {
			initialValue = ev.Data.InitialValue
		}
		fmt.Printf(" %-30s %-15.15s %-40.40s %-10.10v %s%-15.15s%s %s\n",
			key, ev.EventType, ev.Title, dataType, utils.COYellow, initialValue, utils.COReset, ev.Description)
	}

	fmt.Println(utils.CORed + "\nActions:")
	fmt.Println(" ID                             ActionType      Title                                    Arg(s)     Initial Value   Description")
	fmt.Println(" -----------------------------  --------------  ---------------------------------------  ---------  --------------  -----------" + utils.COReset)
	keys = utils.OrderedMapKeys(tdDoc.Actions)
	for _, key := range keys {
		action := tdDoc.Actions[key]
		dataType := "(n/a)"
		initialValue := ""
		if action.Input != nil {
			dataType = action.Input.Type
			initialValue = action.Input.InitialValue
		}
		fmt.Printf(" %-30.30s %-15.15s %-40.40s %-10.10s %s%-15.15s%s %s\n",
			key, action.ActionType, action.Title, dataType, utils.CORed, initialValue, utils.COReset, action.Description)
	}
	fmt.Println()
	return err
}

// HandleListThingVerbose lists a Thing in the directory
func HandleListThingVerbose(hc hubclient.IHubClient, pubID, thingID string) error {
	var rdir directory.IReadDirectory

	rdir = service.NewReadDirectoryService(hc)
	tv, err := rdir.GetTD(pubID, thingID)
	if err != nil {
		return err
	}
	fmt.Println("TD of", pubID, thingID)
	fmt.Printf("%s\n", tv.Data)
	return err
}
