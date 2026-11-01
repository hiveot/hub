package pubsubcli

import (
	"context"
	"fmt"

	"github.com/hiveot/hivekit/go/cells/consumer"
	consumer_recipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	"github.com/urfave/cli/v3"
)

func PubActionCommand(r **consumer_recipe.ConsumerRecipe) *cli.Command {
	return &cli.Command{
		Name:      "pub",
		Usage:     "Publish action for Thing",
		ArgsUsage: "<thingID> <action> [<value>]",
		Description: "Request an action from a Thing, where:\n" +
			"  thingID: ID of the Thing to invoke\n" +
			"  action:  Action to invoke as listed in the Thing's TD document\n" +
			"  value:   Optional value if required by the action",
		Category: "pubsub",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 2 {
				return fmt.Errorf("missing arguments")
			}
			dThingID := cmd.Args().First()
			action := cmd.Args().Get(1)
			args := cmd.Args().Get(2)
			err := HandlePubActions((*r).Consumer, dThingID, action, args)
			return err
		},
	}
}

func HandlePubActions(hc *consumer.Consumer,
	dThingID string, action string, args string) error {

	var reply interface{}
	err := hc.Rpc(dThingID, action, args, &reply, "")
	if err == nil {
		fmt.Printf("Successfully published action '%s' to Thing '%s'\n", action, dThingID)
		if reply != nil {
			fmt.Printf("\n%v\n", reply)
		}
		return nil
	}
	return err
}
