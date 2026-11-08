package pubsubcli

import (
	"context"
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/cells/directory"
	consumer_recipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	"github.com/hiveot/hivekit/go/utils"
	jsoniter "github.com/json-iterator/go"

	"github.com/araddon/dateparse"
	"github.com/urfave/cli/v3"
)

// SubTDCommand shows TD publications
func SubTDCommand(r **consumer_recipe.ConsumerRecipe) *cli.Command {
	return &cli.Command{
		Name:     "subtd",
		Usage:    "SubscribeEvent to TD publications",
		Category: "pubsub",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			err := HandleSubTD((*r).Consumer)
			return err
		},
	}
}

func SubEventsCommand(r **consumer_recipe.ConsumerRecipe) *cli.Command {
	return &cli.Command{
		Name:      "subev",
		Usage:     "SubscribeEvent to Thing events",
		ArgsUsage: "[<thingID> [<key>]]",
		Category:  "pubsub",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			thingID := ""
			key := ""
			if cmd.NArg() > 0 {
				thingID = cmd.Args().Get(0)
			}
			if cmd.NArg() > 1 {
				key = cmd.Args().Get(1)
			}
			if cmd.NArg() > 2 {
				return fmt.Errorf("unexpected arguments")
			}

			err := HandleSubEvents((*r).Consumer, thingID, key)
			return err
		},
	}
}

// HandleSubTD subscribes and prints TD publications
func HandleSubTD(co *consumer.Consumer) error {

	// FIXME: discover to get the directory ID
	err := co.Subscribe(directory.DefaultDirectoryServiceThingID, directory.ThingUpdatedEvent)
	if err != nil {
		return err
	}
	co.SetNotificationHook(func(resp *msg.NotificationMessage) {
		// only look for TD events, ignore directed events
		if resp.Name != directory.ThingUpdatedEvent {
			return
		}

		var tdi wot.TD
		//fmt.Printf("%s\n", event.ValueJSON)
		err := resp.Decode(&tdi)

		if err == nil {
			modifiedTime, _ := dateparse.ParseAny(tdi.Modified) // can be in any TZ
			timeStr := utils.FormatMSE(modifiedTime.UnixMilli(), false)
			fmt.Printf("%-20.20s %-35.35s %-30.30s %-30.30s %-30.30s\n",
				"", resp.ThingID, tdi.Title, tdi.AtType, timeStr)
		}
		return
	})
	fmt.Printf("Sender ID            Thing ID                            Title                          @type                          Timestamp                       \n")
	fmt.Printf("-------------------  ----------------------------------  -----------------------------  -----------------------------  ------------------------------\n")

	time.Sleep(time.Hour * 24)
	return nil
}

// HandleSubEvents subscribes and prints events
func HandleSubEvents(co *consumer.Consumer, thingID string, name string) error {
	fmt.Printf("Subscribing to  thingID: '%s', name: '%s'\n\n", thingID, name)

	fmt.Printf("Time             Agent ID        Thing ID                       Event Name                     Value\n")
	fmt.Printf("---------------  --------------- -----------------------------  -----------------------------  ---------\n")

	err := co.Subscribe(thingID, name)
	co.SetNotificationHook(func(msg *msg.NotificationMessage) {
		createdTime, _ := dateparse.ParseAny(msg.Timestamp)
		timeStr := createdTime.Format("15:04:05.000")

		valueStr := msg.ToString(0)

		//if msg.Name == vocab.EventNameProperties {
		//	var props map[string]interface{}
		//	_ = utils.DecodeAsObject(msg.Data, &props)
		//
		//	valueStr = fmt.Sprintf("%d properties", len(props))
		//
		//	// if its only a single property then show the value
		//	if len(props) == 1 {
		//		for key, val := range props {
		//			valueStr = fmt.Sprintf("{%s=%v}", key, val)
		//		}
		//	}
		//}
		if msg.ThingID == directory.DefaultDirectoryServiceThingID &&
			msg.Name == directory.ThingUpdatedEvent {
			var tdi wot.TD
			tdJSON := msg.ToString(0)
			_ = jsoniter.UnmarshalFromString(tdJSON, &tdi)
			valueStr = fmt.Sprintf("{title:%s, type:%s, nrProps=%d, nrEvents=%d, nrActions=%d}",
				tdi.Title, tdi.AtType, len(tdi.Properties), len(tdi.Events), len(tdi.Actions))
		}

		fmt.Printf("%-16.16s %-15.15s %-30.30s %-30.30s %-40.40s\n",
			timeStr, "", msg.ThingID, msg.Name, valueStr)
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Hour * 24)
	return nil
}
