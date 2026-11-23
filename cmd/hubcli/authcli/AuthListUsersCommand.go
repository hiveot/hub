package authcli

import (
	"context"
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	authn_client "github.com/hiveot/hivekit/go/cells/authn/client"
	corecipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/urfave/cli/v3"
)

// AuthListClientsCommand lists user profiles
func AuthListClientsCommand(r **corecipe.ConsumerRecipe) *cli.Command {
	return &cli.Command{
		Name:     "lu",
		Usage:    "List users",
		Category: "auth",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				cli.ShowSubcommandHelp(cmd)
				err := fmt.Errorf("too many arguments")
				return err
			}
			err := HandleListClients(*r)
			return err
		},
	}
}

// HandleListClients shows a list of user profiles
func HandleListClients(co api.IConsumer) (err error) {

	authnClient := authn_client.NewAuthnAdminClient("", co)

	profileList, err := authnClient.GetProfiles()

	fmt.Println("Users")
	fmt.Println("Login ID             Display Name              Role            Modified")
	fmt.Println("--------             ------------              ----            -------")
	for _, profile := range profileList {
		if profile.Role != api.ClientRoleDevice && profile.Role != api.ClientRoleService {
			fmt.Printf("%-20s %-25s %-15s %s\n",
				profile.ClientID,
				profile.DisplayName,
				profile.Role,
				utils.FormatDateTime(profile.TimeUpdated, ""),
			)
		}
	}
	fmt.Println()
	fmt.Println("Devices/Services")
	fmt.Println("SenderID             Type            Modified")
	fmt.Println("--------             ----            -------")
	for _, profile := range profileList {
		if profile.Role == api.ClientRoleDevice || profile.Role == api.ClientRoleService {
			fmt.Printf("%-20s %-15s %s\n",
				profile.ClientID,
				profile.Role,
				utils.FormatDateTime(profile.TimeUpdated, ""),
			)
		}
	}
	return err
}
