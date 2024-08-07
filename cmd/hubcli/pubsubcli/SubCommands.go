package pubsubcli

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"time"

	"github.com/araddon/dateparse"
	"github.com/urfave/cli/v2"

	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
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
		Name:      "subev",
		Usage:     "Subscribe to Thing events",
		ArgsUsage: "[<thingID> [<key>]]",
		Category:  "pubsub",
		Action: func(cCtx *cli.Context) error {
			thingID := ""
			key := ""
			if cCtx.NArg() > 0 {
				thingID = cCtx.Args().Get(0)
			}
			if cCtx.NArg() > 1 {
				key = cCtx.Args().Get(1)
			}
			if cCtx.NArg() > 2 {
				return fmt.Errorf("Unexpected arguments")
			}

			err := HandleSubEvents(*hc, thingID, key)
			return err
		},
	}
}

// HandleSubTD subscribes and prints TD publications
func HandleSubTD(hc hubclient.IHubClient) error {

	err := hc.Subscribe("", vocab.EventTypeTD)
	if err != nil {
		return err
	}
	hc.SetMessageHandler(func(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
		// only look for TD events, ignore directed events
		if msg.Key != vocab.EventTypeTD {
			return stat.Completed(msg, nil, nil)
		}

		var td things.TD
		//fmt.Printf("%s\n", event.ValueJSON)
		err := msg.Decode(&td)
		if err == nil {
			modifiedTime, _ := dateparse.ParseAny(td.Modified) // can be in any TZ
			timeStr := utils.FormatMSE(modifiedTime.In(time.Local).UnixMilli(), false)
			fmt.Printf("%-20.20s %-35.35s %-30.30s %-30.30s %-30.30s\n",
				msg.SenderID, msg.ThingID, td.Title, td.AtType, timeStr)
		}
		return stat.Completed(msg, nil, nil)
	})
	fmt.Printf("Sender ID            Thing ID                            Title                          @type                          Updated                       \n")
	fmt.Printf("-------------------  ----------------------------------  -----------------------------  -----------------------------  ------------------------------\n")

	time.Sleep(time.Hour * 24)
	return nil
}

// HandleSubEvents subscribes and prints events
func HandleSubEvents(hc hubclient.IHubClient, thingID string, name string) error {
	fmt.Printf("Subscribing to  thingID: '%s', name: '%s'\n\n", thingID, name)

	fmt.Printf("Time             Agent ID        Thing ID                       Event Name                     Value\n")
	fmt.Printf("---------------  --------------- -----------------------------  -----------------------------  ---------\n")

	err := hc.Subscribe(thingID, name)
	hc.SetMessageHandler(func(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
		createdTime, _ := dateparse.ParseAny(msg.Created)
		timeStr := createdTime.Format("15:04:05.000")

		valueStr := msg.DataAsText()

		if msg.Key == vocab.EventTypeProperties {
			var props map[string]interface{}
			_ = msg.Decode(&props)

			valueStr = fmt.Sprintf("%d properties", len(props))

			// if its only a single property then show the value
			if len(props) == 1 {
				for key, val := range props {
					valueStr = fmt.Sprintf("{%s=%v}", key, val)
				}
			}
		} else if msg.Key == vocab.EventTypeTD {
			var td things.TD
			_ = msg.Decode(&td)
			valueStr = fmt.Sprintf("{title:%s, type:%s, nrProps=%d, nrEvents=%d, nrActions=%d}",
				td.Title, td.AtType, len(td.Properties), len(td.Events), len(td.Actions))
		}

		fmt.Printf("%-16.16s %-15.15s %-30.30s %-30.30s %-40.40s\n",
			timeStr, msg.SenderID, msg.ThingID, msg.Key, valueStr)
		return stat.Completed(msg, nil, nil)
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Hour * 24)
	return nil
}
