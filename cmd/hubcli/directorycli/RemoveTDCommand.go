package directorycli

import (
	"context"
	"fmt"

	"github.com/hiveot/hivekit/go/cells/directory"
	consumer_recipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	"github.com/urfave/cli/v3"
)

// Remove a TD from the directory
func DirectoryRemoveTDCommand(r **consumer_recipe.ConsumerRecipe) *cli.Command {
	return &cli.Command{
		Name:      "rmtd",
		Category:  "directory",
		Usage:     "Remove a TD from the directory",
		ArgsUsage: "<thingID> ...",
		Flags:     []cli.Flag{},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dirClient := (*r).GetDirectory()
			var err = fmt.Errorf("missing ThingID parameters")
			if cmd.NArg() == 0 {
				cli.ShowSubcommandHelp(cmd)
			} else {
				thingIDs := cmd.Args().Slice()
				err = HandleRemoveTD(dirClient, thingIDs)
			}
			return err
		},
	}
}

// HandleRemoveTD removes a TD from the directory
func HandleRemoveTD(dirClient directory.IDirectoryClient, thingIDs []string) (err error) {

	// todo: some checking and ask for confirmation

	for _, thingID := range thingIDs {
		err = dirClient.DeleteThing(thingID)
		if err == nil {
			fmt.Printf("Thing '%s' was removed from the directory\n", thingID)
		} else {
			fmt.Printf("Error removing '%s' from the directory: %s\n", thingID, err.Error())
			break
		}
	}
	return err
}
