package pubsubcli

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/vocab"
	"time"

	"github.com/araddon/dateparse"
	"github.com/urfave/cli/v2"

	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/thing"
)

// SubTDCommand shows TD publications
func SubTDCommand(hc *hubclient.IHubClient) *cli.Command {
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

func SubEventsCommand(hc *hubclient.IHubClient) *cli.Command {
	return &cli.Command{
		Name:     "subev",
		Usage:    "Subscribe to Thing events",
		Category: "pubsub",
		Action: func(cCtx *cli.Context) error {
			err := HandleSubEvents(*hc)
			return err
		},
	}
}

// HandleSubTD subscribes and prints TD publications
func HandleSubTD(hc hubclient.IHubClient) error {

	sub, err := hc.SubEvents("", "", vocab.EventNameTD,
		func(msg *thing.ThingValue) {
			var td thing.TD
			//fmt.Printf("%s\n", event.ValueJSON)
			err := json.Unmarshal(msg.Data, &td)
			if err == nil {
				modifiedTime, _ := dateparse.ParseAny(td.Modified)                  // can be in any TZ
				timeStr := modifiedTime.In(time.Local).Format("15:04:05.000 -0700") // want local time
				fmt.Printf("%-20s %-25s %-30s %-20s %-18s\n",
					msg.AgentID, msg.ThingID, td.Title, td.DeviceType, timeStr)
			}
		})
	defer sub.Unsubscribe()
	fmt.Printf("Agent ID             Thing ID                  Title                          Type                 Modified          \n")
	fmt.Printf("-------------------  ------------------------  -----------------------------  -------------------  ------------------\n")

	if err != nil {
		return err
	}
	time.Sleep(time.Hour * 24)
	return nil
}

// HandleSubEvents subscribes and prints value and property events
func HandleSubEvents(hc hubclient.IHubClient) error {
	fmt.Printf("Time             AgentID              ThingID                   Event Name                     Value\n")
	fmt.Printf("---------------  -------------------  ------------------------  -----------------------------  ---------\n")

	sub, err := hc.SubEvents("", "", "",
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
