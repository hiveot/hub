package pubsubcli

import (
	"fmt"
	"github.com/urfave/cli/v2"

	"github.com/hiveot/hub/lib/hubclient"
)

func PubActionCommand(hc *hubclient.IHubClient) *cli.Command {
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
			err := HandlePubActions(*hc, pubID, thingID, action, args)
			return err
		},
	}
}

func HandlePubActions(hc hubclient.IHubClient,
	pubID string, thingID string, action string, args string) error {

	ar, err := hc.PubAction(pubID, thingID, action, []byte(args))
	_ = ar
	if err == nil {
		fmt.Printf("Successfully published action '%s' to thing '%s'\n", action, thingID)
	}
	return err
}
