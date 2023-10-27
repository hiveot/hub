package pubsubcli

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/lib/vocab"
	"time"

	"github.com/araddon/dateparse"
	"github.com/urfave/cli/v2"

	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/thing"
)

// SubTDCommand shows TD publications
func SubTDCommand(hc **hubclient.HubClient) *cli.Command {
	return &cli.Command{
		Name:     "subtd",
		Usage:    "Subscribe to TD publications",
		Category: "pubsub",
		Action: func(cCtx *cli.Context) error {
			err := HandleSubTD(*hc)
			return err
		},
	}
}

func SubEventsCommand(hc **hubclient.HubClient) *cli.Command {
	return &cli.Command{
		Name:      "subev",
		Usage:     "Subscribe to Thing events",
		ArgsUsage: "[<agentID> [<thingID>]]",
		Category:  "pubsub",
		Action: func(cCtx *cli.Context) error {
			agentID := ""
			thingID := ""
			name := ""
			if cCtx.NArg() > 0 {
				agentID = cCtx.Args().Get(0)
			}
			if cCtx.NArg() > 1 {
				thingID = cCtx.Args().Get(1)
			}
			if cCtx.NArg() > 2 {
				name = cCtx.Args().Get(2)
			}
			if cCtx.NArg() > 3 {
				return fmt.Errorf("Unexpected arguments")
			}

			err := HandleSubEvents(*hc, agentID, thingID, name)
			return err
		},
	}
}

// HandleSubTD subscribes and prints TD publications
func HandleSubTD(hc *hubclient.HubClient) error {

	sub, err := hc.SubEvents("", "", vocab.EventNameTD,
		func(msg *thing.ThingValue) {
			var td thing.TD
			//fmt.Printf("%s\n", event.ValueJSON)
			err := json.Unmarshal(msg.Data, &td)
			if err == nil {
				modifiedTime, _ := dateparse.ParseAny(td.Modified) // can be in any TZ
				timeStr := utils.FormatMSE(modifiedTime.In(time.Local).UnixMilli(), false)
				fmt.Printf("%-20s %-25s %-30s %-20s %-18s\n",
					msg.AgentID, msg.ThingID, td.Title, td.DeviceType, timeStr)
			}
		})
	defer sub.Unsubscribe()
	fmt.Printf("Agent ID             Thing ID                  Title                          Type                 Updated           \n")
	fmt.Printf("-------------------  ------------------------  -----------------------------  -------------------  --------------------\n")

	if err != nil {
		return err
	}
	time.Sleep(time.Hour * 24)
	return nil
}

// HandleSubEvents subscribes and prints value and property events
func HandleSubEvents(hc *hubclient.HubClient, agentID string, thingID string, name string) error {
	fmt.Printf("Subscribing to agentID: '%s', thingID: '%s', name: '%s'\n\n", agentID, thingID, name)

	fmt.Printf("Time             Agent ID             Thing ID                  Event Name                     Value\n")
	fmt.Printf("---------------  -------------------  ------------------------  -----------------------------  ---------\n")

	sub, err := hc.SubEvents(agentID, thingID, name,
		func(msg *thing.ThingValue) {
			createdTime := time.UnixMilli(msg.CreatedMSec)
			timeStr := createdTime.Format("15:04:05.000")
			value := fmt.Sprintf("%-.30s", msg.Data)
			if msg.Name == vocab.EventNameProps {
				var props map[string]interface{}
				_ = json.Unmarshal(msg.Data, &props)
				value = fmt.Sprintf("%d properties", len(props))
			} else if msg.Name == vocab.EventNameTD {
				var td thing.TD
				_ = json.Unmarshal(msg.Data, &td)
				value = fmt.Sprintf("{title:%s, type:%s, nrProps=%d, nrEvents=%d, nrActions=%d}",
					td.Title, td.DeviceType, len(td.Properties), len(td.Events), len(td.Actions))
			}

			fmt.Printf("%-16s %-20s %-25s %-30s %-30s\n",
				timeStr, msg.AgentID, msg.ThingID, msg.Name, value)
		})
	defer sub.Unsubscribe()
	if err != nil {
		return err
	}
	time.Sleep(time.Hour * 24)
	return nil
}
