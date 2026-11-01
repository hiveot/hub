package authcli

import (
	"context"
	"fmt"
	"strings"

	"github.com/hiveot/hivekit/go/api"
	authn_client "github.com/hiveot/hivekit/go/cells/authn/client"
	"github.com/hiveot/hivekit/go/cells/consumer"
	consumer_recipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	"github.com/urfave/cli/v3"
)

// AuthSetRoleCommand changes a user's role
func AuthSetRoleCommand(r **consumer_recipe.ConsumerRecipe) *cli.Command {
	validRoles := []string{
		string(api.ClientRoleViewer), string(api.ClientRoleOperator),
		string(api.ClientRoleManager), string(api.ClientRoleAdmin),
	}

	return &cli.Command{
		Name:      "setrole",
		Usage:     "Set a new role",
		ArgsUsage: "<loginID> <newrole>",
		UsageText: "Valid roles: " + strings.Join(validRoles, ", "),
		Category:  "auth",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 2 {
				err := fmt.Errorf("expected 2 arguments")
				return err
			}
			loginID := cmd.Args().Get(0)
			newRole := cmd.Args().Get(1)
			err := HandleSetRole(*r, loginID, newRole)
			return err
		},
	}
}

// HandleSetRole sets a new role
//
//	loginID is the ID or email of the user
//	newPassword can be empty to auto-generate a password
func HandleSetRole(co consumer.IConsumer, loginID string, newRole string) error {

	authnClient := authn_client.NewAuthnAdminClient("", co)
	prof, err := authnClient.GetProfile(loginID)
	if err != nil {
		return err
	}
	prof.Role = newRole
	err = authnClient.UpdateClientProfile(prof)
	if err != nil {
		//fmt.Println("Error: " + err.Error())
	} else {
		fmt.Println("User " + loginID + " role has been updated to " + newRole)
	}
	return err
}
