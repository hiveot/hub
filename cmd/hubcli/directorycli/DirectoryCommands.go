package directorycli

import (
	"encoding/json"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/core/directory/dirclient"
	"github.com/hiveot/hub/core/history/historyclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"github.com/urfave/cli/v2"
	"log/slog"
	"time"

	"github.com/hiveot/hub/lib/hubclient"
)

func DirectoryListCommand(hc **hubclient.HubClient) *cli.Command {
	var verbose = false
	return &cli.Command{
		Name:      "ld",
		Category:  "directory",
		Usage:     "List directory of Things",
		ArgsUsage: "[<agentID> <thingID>]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "raw",
				Usage:       "raw, display raw json",
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
func HandleListDirectory(hc *hubclient.HubClient) (err error) {
	offset := 0
	limit := 100
	rdir := dirclient.NewReadDirectoryClient(hc)

	cursor, err := rdir.GetCursor()
	if err != nil {
		return err
	}
	fmt.Printf("Agent ID / Thing ID                 @type                               Title                                #props  #events #actions   Updated         \n")
	fmt.Printf("----------------------------------  ----------------------------------  -----------------------------------  ------  ------- --------   -----------------------------\n")
	i := 0
	tv, valid, err := cursor.First()
	if offset > 0 {
		// TODO, skip
		//tv, valid = cursor.Skip(offset)
	}
	for ; valid && i < limit; tv, valid, err = cursor.Next() {
		var tdDoc things.TD
		err = json.Unmarshal(tv.Data, &tdDoc)
		var utime time.Time
		if tdDoc.Modified != "" {
			utime, err = dateparse.ParseAny(tdDoc.Modified)
		} else if tdDoc.Created != "" {
			utime, err = dateparse.ParseAny(tdDoc.Created)
		}
		//timeStr := utime.In(time.Local).Format("02 Jan 2006 15:04:05 -0700")
		timeStr := utils.FormatMSE(utime.In(time.Local).UnixMilli(), false)

		fmt.Printf("%-35s %-35.35s %-35.35s %7d  %7d  %7d   %-30s\n",
			tv.AgentID+" / "+tdDoc.ID,
			tdDoc.AtType,
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
func HandleListThing(hc *hubclient.HubClient, pubID, thingID string) error {
	var tdDoc things.TD

	rdir := dirclient.NewReadDirectoryClient(hc)
	rhist := historyclient.NewReadHistoryClient(hc)
	tv, err := rdir.GetTD(pubID, thingID)
	if err != nil {
		return err
	}
	err = json.Unmarshal(tv.Data, &tdDoc)
	if err != nil {
		return err
	}
	histValues, err := rhist.GetLatest(pubID, thingID, nil)

	if err != nil {
		slog.Error("Unable to read history:", "err", err)
	}
	fmt.Printf("%sTD of %s %s:%s\n", utils.COBlue, pubID, thingID, utils.COReset)
	fmt.Printf(" title:       %s\n", tdDoc.Title)
	fmt.Printf(" description: %s\n", tdDoc.Description)
	fmt.Printf(" @type:       %s\n", tdDoc.AtType)
	fmt.Printf(" modified:    %s\n", tdDoc.Modified)
	fmt.Println("")

	fmt.Println(utils.COGreen + "Attributes:")
	fmt.Println(" ID                             Title                                    Value           Description")
	fmt.Println(" ----------------------------   ---------------------------------------  -------------   -----------" + utils.COReset)
	keys := utils.OrderedMapKeys(tdDoc.Properties)
	for _, key := range keys {
		prop, found := tdDoc.Properties[key]
		if found && prop.ReadOnly {
			value := histValues.ToString(key)
			fmt.Printf(" %-30s %-40.40s %s%-15.15v%s %.80s\n",
				key, prop.Title, utils.COGreen, value, utils.COReset, prop.Description)
		}
	}
	fmt.Println()
	fmt.Println(utils.COBlue + "Configuration:")
	fmt.Println(" ID                             Title                                    DataType   Value                Description")
	fmt.Println(" -----------------------------  ---------------------------------------  ---------  ------------------   -----------" + utils.COReset)
	for _, key := range keys {
		prop, found := tdDoc.Properties[key]
		if found && !prop.ReadOnly {
			value := histValues.ToString(key)
			fmt.Printf(" %-30s %-40.40s %-10s %s%-20.20v%s %.80s\n",
				key, prop.Title, prop.Type, utils.COBlue, value, utils.COReset, prop.Description)
		}
	}

	fmt.Println(utils.COYellow + "\nEvents:")
	fmt.Println(" ID                             EventType                 Title                                    DataType   Value           Description")
	fmt.Println(" -----------------------------  ------------------------  ---------------------------------------  ---------  --------------  -----------" + utils.COReset)
	keys = utils.OrderedMapKeys(tdDoc.Events)
	for _, key := range keys {
		ev := tdDoc.Events[key]
		dataType := "(n/a)"
		if ev.Data != nil {
			dataType = ev.Data.Type
		}
		value := histValues.ToString(key)
		if ev.Data != nil {
			//initialValue = ev.Data.InitialValue
		}
		fmt.Printf(" %-30s %-25.25s %-40.40s %-10.10v %s%-15.15s%s %.80s\n",
			key, ev.EventType, ev.Title, dataType, utils.COYellow, value, utils.COReset, ev.Description)
	}

	fmt.Println(utils.CORed + "\nActions:")
	fmt.Println(" ID                             ActionType                Title                                    Arg(s)     Value           Description")
	fmt.Println(" -----------------------------  ------------------------  ---------------------------------------  ---------  --------------  -----------" + utils.COReset)
	keys = utils.OrderedMapKeys(tdDoc.Actions)
	for _, key := range keys {
		action := tdDoc.Actions[key]
		dataType := "(n/a)"
		value := histValues.ToString(key)
		if action.Input != nil {
			dataType = action.Input.Type
			//initialValue = action.Input.InitialValue
		}
		fmt.Printf(" %-30.30s %-25.25s %-40.40s %-10.10s %s%-15.15s%s %.80s\n",
			key, action.ActionType, action.Title, dataType, utils.CORed, value, utils.COReset, action.Description)
	}
	fmt.Println()
	return err
}

// HandleListThingVerbose lists a Thing in the directory
func HandleListThingVerbose(hc *hubclient.HubClient, pubID, thingID string) error {
	rdir := dirclient.NewReadDirectoryClient(hc)
	tv, err := rdir.GetTD(pubID, thingID)
	if err != nil {
		return err
	}
	fmt.Println("TD of", pubID, thingID)
	fmt.Printf("%s\n", tv.Data)
	return err
}
