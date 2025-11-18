package pubsubcli

import (
	"fmt"

	"github.com/hiveot/hivekit/go/consumer"
	"github.com/urfave/cli/v2"
)

func PubActionCommand(hc **consumer.Consumer) *cli.Command {
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
