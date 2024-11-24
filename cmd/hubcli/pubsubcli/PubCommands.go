package pubsubcli

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/wot/protocolclients"
	"github.com/urfave/cli/v2"
)

func PubActionCommand(hc *clients.IConsumer) *cli.Command {
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

func HandlePubActions(hc clients.IConsumer,
	dThingID string, action string, args string) error {

	var reply interface{}
	stat := hc.InvokeAction(dThingID, action, args, &reply, "")
	if stat.Error == "" {
		fmt.Printf("Successfully published action '%s' to Thing '%s'\n", action, dThingID)
		if reply != nil {
			fmt.Printf("\n%v\n", reply)
		}
		return nil
	}
	return errors.New(stat.Error)
}
