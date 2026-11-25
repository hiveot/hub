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

	fmtString := "%-20s %-25s %-15s %-26s %-15s\n"
	fmt.Println("Users")
	fmt.Printf(fmtString, "Login ID", "Display Name", "Role", "Created", "Last Login")
	fmt.Printf(fmtString, "--------", "------------", "----", "-------", "----------")

	for _, profile := range profileList {
		if profile.Role != api.ClientRoleDevice && profile.Role != api.ClientRoleService {
			clientID := profile.ClientID
			if profile.Disabled {
				clientID = clientID + "(disabled)"
			}
			fmt.Printf(fmtString,
				clientID,
				profile.DisplayName,
				profile.Role,
				utils.FormatDateTime(profile.TimeCreated, ""),
				utils.FormatDateTime(profile.LastLogin, "S"),
			)
		}
	}
	fmt.Println()
	fmt.Println("Devices/Services")
	fmtString = "%-20s %-20s %-15s %-26s %-15s\n"
	fmt.Printf(fmtString, "Client ID", "Display Name", "Type", "Created", "Modified")
	fmt.Printf(fmtString, "---------", "------------", "----", "-------", "--------")
	for _, profile := range profileList {
		if profile.Role == api.ClientRoleDevice || profile.Role == api.ClientRoleService {
			clientID := profile.ClientID
			if profile.Disabled {
				clientID = clientID + "(disabled)"
			}
			fmt.Printf(fmtString,
				clientID,
				profile.DisplayName,
				profile.Role,
				utils.FormatDateTime(profile.TimeCreated, ""),
				utils.FormatDateTime(profile.TimeUpdated, ""),
			)
		}
	}
	return err
}
