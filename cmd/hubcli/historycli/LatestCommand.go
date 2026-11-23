package historycli

import (
	"context"
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	consumer_recipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/urfave/cli/v3"
)

func LatestValuesCommand(r **consumer_recipe.ConsumerRecipe) *cli.Command {
	return &cli.Command{
		Name:      "latest",
		Usage:     "Show the latest values of a thing",
		ArgsUsage: "<thingID> [affordanceName]",
		Category:  "history",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				cli.ShowSubcommandHelp(cmd)
				return fmt.Errorf("thingID and optional name expected")
			}
			thingID := cmd.Args().First()
			name := ""
			if cmd.NArg() >= 2 {
				name = cmd.Args().Get(1)
			}
			err := HandleLatestValues(*r, thingID, name)
			return err
		},
	}
}

func HandleLatestValues(
	co api.IConsumer, thingID string, name string) error {
	var err error

	var propValues map[string]msg.ThingValue
	var eventValues map[string]msg.ThingValue

	if name != "" {
		tv, err := co.ReadProperty(thingID, name)
		if err == nil {
			propValues = map[string]msg.ThingValue{name: tv}
		}
		tv, err = co.ReadEvent(thingID, name)
		if err == nil {
			eventValues = map[string]msg.ThingValue{name: tv}
		}
	} else {
		propValues, err = co.ReadAllProperties(thingID)
		eventValues, err = co.ReadAllEvents(thingID)
	}

	fmt.Println("Values of Thing:", thingID)
	fmt.Println("Prop Name                          Value                    Last Update")
	fmt.Println("---------                          ------                   ----------- ")
	propKeys := utils.OrderedMapKeys(propValues)
	for _, propName := range propKeys {
		tv := propValues[propName]
		fmt.Printf("%-34.34s %-24.24s %-20.20s \n",
			propName,
			fmt.Sprintf("%.32s", tv.ToString(30)),
			//utime.Format("02 Jan 2006 15:04:05 -0700"),
			utils.FormatDateTime(tv.Timestamp),
		)
	}

	if len(eventValues) > 0 {
		fmt.Println()
		fmt.Println("Event Name                         Value                    Last Update")
		fmt.Println("-----------                        ------                   ----------- ")
		evKeys := utils.OrderedMapKeys(eventValues)
		for _, evName := range evKeys {
			tv := eventValues[evName]
			fmt.Printf("%-34.34s %-24.24s %-24.24s \n",
				evName,
				fmt.Sprintf("%.32s", tv.ToString(30)),
				//utime.Format("02 Jan 2006 15:04:05 -0700"),
				utils.FormatDateTime(tv.Timestamp),
			)
		}
	}
	return err
}
