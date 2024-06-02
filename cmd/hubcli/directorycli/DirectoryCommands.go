package directorycli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/api/go/directory"
	"github.com/hiveot/hub/api/go/outbox"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"github.com/urfave/cli/v2"
	"log/slog"
	"time"

	"github.com/hiveot/hub/lib/hubclient"
)

func DirectoryListCommand(hc *hubclient.IHubClient) *cli.Command {
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
				err = HandleListDirectory(*hc)
			} else if cCtx.NArg() == 1 {
				if !verbose {
					err = HandleListThing(*hc, cCtx.Args().First())
				} else {
					err = HandleListThingVerbose(*hc, cCtx.Args().First())
				}
			}
			return err
		},
	}
}

// HandleListDirectory lists the directory content
func HandleListDirectory(hc hubclient.IHubClient) (err error) {
	// todo: iterate with offset and limit
	args := directory.ReadTDsArgs{Offset: 0, Limit: 300}
	resp, err := directory.ReadTDs(hc, args)
	if err != nil {
		return err
	}
	fmt.Printf("Thing ID                            @type                               Title                                #props  #events #actions   GetUpdated         \n")
	fmt.Printf("----------------------------------  ----------------------------------  -----------------------------------  ------  ------- --------   -----------------------------\n")
	for _, tdJSON := range resp.Output {
		var tdDoc things.TD
		err = json.Unmarshal([]byte(tdJSON), &tdDoc)
		var utime time.Time
		if tdDoc.Modified != "" {
			utime, err = dateparse.ParseAny(tdDoc.Modified)
		} else if tdDoc.Created != "" {
			utime, err = dateparse.ParseAny(tdDoc.Created)
		}
		//timeStr := utime.In(time.Local).Format("02 Jan 2006 15:04:05 -0700")
		timeStr := utils.FormatMSE(utime.In(time.Local).UnixMilli(), false)

		fmt.Printf("%-35s %-35.35s %-35.35s %7d  %7d  %7d   %-30s\n",
			tdDoc.ID,
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
func HandleListThing(hc hubclient.IHubClient, thingID string) error {
	var tdDoc things.TD

	tdResp, err := directory.ReadTD(hc,
		directory.ReadTDArgs{ThingID: thingID})
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(tdResp.Output), &tdDoc)
	if err != nil {
		return err
	}
	latestResp, err := outbox.ReadLatest(hc,
		outbox.ReadLatestArgs{ThingID: thingID})
	valueMap := things.ThingMessageMap{}
	err = json.Unmarshal([]byte(latestResp.Values), &valueMap)

	if err != nil {
		slog.Error("Unable to read history:", "err", err)
	}
	fmt.Printf("%sTD of %s    %s\n", utils.COBlue, thingID, utils.COReset)
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
			value := valueMap.ToString(key)
			fmt.Printf(" %-30s %-40.40s %s%-15.15v%s %-.80s\n",
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
			value := valueMap.ToString(key)
			fmt.Printf(" %-30s %-40.40s %-10.10s %s%-15.15s%s %-.80s\n",
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
		value := valueMap.ToString(key)
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
		value := valueMap.ToString(key)
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

// HandleListThingVerbose lists a Thing full TD
func HandleListThingVerbose(hc hubclient.IHubClient, thingID string) error {
	resp, err := directory.ReadTD(hc, directory.ReadTDArgs{ThingID: thingID})

	if err != nil {
		return err
	}
	fmt.Println("TD of", thingID)
	var buf bytes.Buffer
	_ = json.Indent(&buf, []byte(resp.Output), "", "\t")
	fmt.Printf("%s\n", buf.Bytes())
	return err
}
