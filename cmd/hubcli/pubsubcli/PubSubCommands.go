package pubsubcli

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/hubapi"
	"time"

	"github.com/araddon/dateparse"
	"github.com/urfave/cli/v2"

	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/pkg/pubsub"
	"github.com/hiveot/hub/pkg/pubsub/capnpclient"
)

func PubActionCommand(ctx context.Context, runFolder *string) *cli.Command {
	return &cli.Command{
		Name:      "pub",
		Usage:     "Publish action for Thing",
		ArgsUsage: "<pubID> <thingID> <action> [<value>]",
		Description: "Request an action from a Thing, where:\n" +
			"  pubID:   ID of the publisher of the Thing as shown by 'hubapi ld'\n" +
			"  thingID: ID of the Thing to invoke\n" +
			"  action:  Action to invoke as listed in the Thing's TD document\n" +
			"  value:   Optional value if required by the action",
		Category: "pubsub",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() < 3 {
				return fmt.Errorf("missing arguments")
			}
			pubID := cCtx.Args().First()
			thingID := cCtx.Args().Get(1)
			action := cCtx.Args().Get(2)
			args := cCtx.Args().Get(3)
			err := HandlePubActions(ctx, *runFolder, pubID, thingID, action, args)
			return err
		},
	}
}

// SubTDCommand shows TD publications
func SubTDCommand(ctx context.Context, runFolder *string) *cli.Command {
	return &cli.Command{
		Name:     "subtd",
		Usage:    "Subscribe to TD publications",
		Category: "pubsub",
		Action: func(cCtx *cli.Context) error {
			err := HandleSubTD(ctx, *runFolder)
			return err
		},
	}
}

func SubEventsCommand(ctx context.Context, runFolder *string) *cli.Command {
	return &cli.Command{
		Name:     "subev",
		Usage:    "Subscribe to Thing events",
		Category: "pubsub",
		Action: func(cCtx *cli.Context) error {
			err := HandleSubEvents(ctx, *runFolder)
			return err
		},
	}
}

func HandlePubActions(ctx context.Context, runFolder string, pubID string, thingID string, action string, args string) error {
	var pubSubSvc pubsub.IPubSubService

	capClient, err := hubclient.ConnectWithCapnpUDS(pubsub.ServiceName, runFolder)
	if err == nil {
		pubSubSvc = capnpclient.NewPubSubCapnpClient(capClient)
	}
	if err != nil {
		return err
	}
	pubSubUser, _ := pubSubSvc.CapUserPubSub(ctx, "hubcli")
	err = pubSubUser.PubAction(ctx, pubID, thingID, action, []byte(args))
	if err == nil {
		fmt.Printf("Successfully published action '%s' to thing '%s'\n", action, thingID)
	}
	return err
}

// HandleSubTD subscribes and prints TD publications
func HandleSubTD(ctx context.Context, runFolder string) error {
	var pubSubSvc pubsub.IPubSubService

	capClient, err := hubclient.ConnectWithCapnpUDS(pubsub.ServiceName, runFolder)
	if err == nil {
		pubSubSvc = capnpclient.NewPubSubCapnpClient(capClient)
	}
	if err != nil {
		return err
	}
	pubSubUser, _ := pubSubSvc.CapUserPubSub(ctx, "hubcli")
	err = pubSubUser.SubEvent(ctx, "", "", hubapi.EventNameTD, func(event thing.ThingValue) {
		var td thing.TD
		//fmt.Printf("%s\n", event.ValueJSON)
		err = json.Unmarshal(event.Data, &td)

		modifiedTime, _ := dateparse.ParseAny(td.Modified)                  // can be in any TZ
		timeStr := modifiedTime.In(time.Local).Format("15:04:05.000 -0700") // want local time
		fmt.Printf("%-20s %-25s %-30s %-20s %-18s\n",
			event.PublisherID, event.ThingID, td.Title, td.DeviceType, timeStr)
	})
	fmt.Printf("Publisher ID         Thing ID                  Title                          Type                 Modified          \n")
	fmt.Printf("-------------------  ------------------------  -----------------------------  -------------------  ------------------\n")

	if err != nil {
		return err
	}
	time.Sleep(time.Hour * 24)
	return nil
}

// HandleSubEvents subscribes and prints value and property events
func HandleSubEvents(ctx context.Context, runFolder string) error {
	var pubSubSvc pubsub.IPubSubService

	capClient, err := hubclient.ConnectWithCapnpUDS(pubsub.ServiceName, runFolder)
	if err == nil {
		pubSubSvc = capnpclient.NewPubSubCapnpClient(capClient)
	}
	if err != nil {
		return err
	}
	fmt.Printf("Time             Publisher            ThingID                   EventID                        Value\n")
	fmt.Printf("---------------  -------------------  ------------------------  -----------------------------  ---------\n")

	pubSubUser, _ := pubSubSvc.CapServicePubSub(ctx, "hubcli")
	err = pubSubUser.SubEvents(ctx, "", "", "", func(event thing.ThingValue) {
		createdTime, _ := dateparse.ParseAny(event.Created)
		timeStr := createdTime.Format("15:04:05.000")
		value := fmt.Sprintf("%-.30s", event.Data)
		if event.ID == hubapi.EventNameProperties {
			var props map[string]interface{}
			_ = json.Unmarshal(event.Data, &props)
			value = fmt.Sprintf("%d properties", len(props))
		} else if event.ID == hubapi.EventNameTD {
			var td thing.TD
			_ = json.Unmarshal(event.Data, &td)
			value = fmt.Sprintf("{title:%s, type:%s, nrProps=%d, nrEvents=%d, nrActions=%d}",
				td.Title, td.DeviceType, len(td.Properties), len(td.Events), len(td.Actions))
		}

		fmt.Printf("%-16s %-20s %-25s %-30s %-30s\n",
			timeStr, event.PublisherID, event.ThingID, event.ID, value)
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Hour * 24)
	return nil
}
