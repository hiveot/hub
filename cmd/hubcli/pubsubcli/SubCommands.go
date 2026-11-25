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
	jsoniter "github.com/json-iterator/go"

	"github.com/araddon/dateparse"
	"github.com/urfave/cli/v3"
)

func SubValueChanges(r **consumer_recipe.ConsumerRecipe) *cli.Command {
	return &cli.Command{
		Name:      "sub",
		Usage:     "Subscribe to Thing events and property changes",
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
			dirCl := (*r).GetDirectory()
			err := HandleSubEvProp((*r).Consumer, dirCl, thingID, key)
			return err
		},
	}
}

// HandleSubEvProp subscribes to event and property changes
func HandleSubEvProp(co *consumer.Consumer, dirCl directory.IDirectoryClient, thingID string, name string) error {
	fmt.Printf("Subscribing to thingID: '%s', name: '%s'\n\n", thingID, name)

	fmt.Printf("Time             Sender          Title (Thing ID)                    Type          Affordance Name                Value\n")
	fmt.Printf("---------------  --------------- -----------------------------       ------------- -----------------------------  ---------\n")

	err := co.Subscribe(thingID, name)
	err = co.ObserveProperty(thingID, name)
	co.SetNotificationHook(func(msg *msg.NotificationMessage) {
		var tdoc *wot.TD

		tdoc, _ = dirCl.GetTD(msg.ThingID)
		createdTime, _ := dateparse.ParseAny(msg.Timestamp)
		timeStr := createdTime.Local().Format("15:04:05.000")

		// FIXME: json unmarshals integers as float
		valueStr := msg.ToString(0)

		// Make a TD presentable
		if msg.ThingID == directory.DirectoryServiceCellType &&
			msg.Name == directory.ThingUpdatedEvent {
			var tdoc2 wot.TD
			tdJSON := msg.ToString(0)
			_ = jsoniter.UnmarshalFromString(tdJSON, &tdoc2)
			valueStr = fmt.Sprintf("{ID: %s, title:%s, nrProps=%d, nrEvents=%d, nrActions=%d}",
				tdoc2.ID, tdoc2.Title, len(tdoc2.Properties), len(tdoc2.Events), len(tdoc2.Actions))
		}

		title := "?"
		if tdoc != nil {
			title = tdoc.Title
		}
		thingIDTitle := fmt.Sprintf("%s (%s)", title, msg.ThingID)

		fmt.Printf("%-16.16s %-15.15s %-35.35s %-13.13s %-30.30s %-40.40s\n",
			timeStr, msg.SenderID, thingIDTitle, msg.AffordanceType, msg.Name, valueStr)
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Hour * 24)
	return nil
}
