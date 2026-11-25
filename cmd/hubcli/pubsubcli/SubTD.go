package pubsubcli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/cells/directory"
	consumer_recipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	"github.com/hiveot/hivekit/go/utils"

	"github.com/araddon/dateparse"
	"github.com/urfave/cli/v3"
)

// SubTDCommand shows TD publications
func SubTDCommand(r **consumer_recipe.ConsumerRecipe) *cli.Command {
	return &cli.Command{
		Name:     "subtd",
		Usage:    "Subscribe to TD update events",
		Category: "pubsub",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dirTD := (*r).GetEnv().GetDirTD()
			err := HandleSubTD((*r).Consumer, dirTD)
			return err
		},
	}
}

// HandleSubTD subscribes and prints TD publications
func HandleSubTD(co *consumer.Consumer, dirTD *wot.TD) error {

	// If a directory is provided, use its TD ID to subscribe
	dirID := directory.DirectoryServiceCellType
	if dirTD != nil {
		dirID = dirTD.ID
	}
	err := co.Subscribe(dirID, directory.ThingUpdatedEvent)
	if err != nil {
		return err
	}
	fmtString := "%-20.20s %-35.35s %-30.30s %-30.30s %-30.30s\n"
	co.SetNotificationHook(func(resp *msg.NotificationMessage) {
		// slog.Info("HandleSubTD: received notification",
		// 	"thingID", resp.ThingID,
		// 	"name", resp.Name)

		// only look for TD events, ignore directed events
		if resp.Name != directory.ThingUpdatedEvent {
			return
		}

		var tdJSON string
		var tdoc *wot.TD
		err := resp.Decode(&tdJSON)
		tdoc, err = wot.UnmarshalTD(tdJSON)
		if err != nil {
			fmt.Printf("Unable to decode TD from %s: %s\n", resp.ThingID, err.Error())
			return
		}
		modifiedTime, _ := dateparse.ParseAny(tdoc.Modified) // can be in any TZ
		timeStr := utils.FormatMSE(modifiedTime.UnixMilli(), false)
		atType := strings.Join(tdoc.GetAtTypes(), ",")
		fmt.Printf(fmtString, resp.SenderID, resp.ThingID, tdoc.Title, atType, timeStr)

	})
	fmt.Printf(fmtString, "Sender ID", "Thing ID", "Title", "@type", "Timestamp")
	fmt.Printf(fmtString, "---------", "--------", "-----", "-----", "---------")

	time.Sleep(time.Hour * 24)
	return nil
}
