package pubsubcli

import (
	"errors"
	"fmt"
	"github.com/urfave/cli/v2"

	"github.com/hiveot/hub/lib/hubclient"
)

func PubActionCommand(hc *hubclient.IConsumerClient) *cli.Command {
	return &cli.Command{
		Name:      "pub",
		Usage:     "Publish action for Thing",
		ArgsUsage: "<thingID> <action> [<value>]",
		Description: "Request an action from a Thing, where:\n" +
			"  thingID: ID of the Thing to invoke\n" +
			"  action:  Action to invoke as listed in the Thing's TD document\n" +
			"  value:   Optional value if required by the action",
		Category: "pubsub",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() < 2 {
				return fmt.Errorf("missing arguments")
			}
			dThingID := cCtx.Args().First()
			action := cCtx.Args().Get(1)
			args := cCtx.Args().Get(2)
			err := HandlePubActions(*hc, dThingID, action, args)
			return err
		},
	}
}

func HandlePubActions(hc hubclient.IConsumerClient,
	dThingID string, action string, args string) error {

	stat := hc.InvokeAction(dThingID, action, args, "")
	if stat.Error == "" {
		fmt.Printf("Successfully published action '%s' to Thing '%s'\n", action, dThingID)
		return nil
	}
	return errors.New(stat.Error)
}
