package authcli

import (
	"fmt"
	"strings"

	"github.com/hiveot/hivekit/go/modules/authn"
	authnpkg "github.com/hiveot/hivekit/go/modules/authn/pkg"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/urfave/cli/v2"
)

// AuthSetRoleCommand changes a user's role
func AuthSetRoleCommand(hc **consumer.Consumer) *cli.Command {
	validRoles := []string{
		string(authn.ClientRoleViewer), string(authn.ClientRoleOperator),
		string(authn.ClientRoleManager), string(authn.ClientRoleAdmin),
	}

	return &cli.Command{
		Name:      "setrole",
		Usage:     "Set a new role",
		ArgsUsage: "<loginID> <newrole>",
		UsageText: "Valid roles: " + strings.Join(validRoles, ", "),
		Category:  "auth",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 2 {
				err := fmt.Errorf("expected 2 arguments")
				return err
			}
			loginID := cCtx.Args().Get(0)
			newRole := cCtx.Args().Get(1)
			err := HandleSetRole(*hc, loginID, newRole)
			return err
		},
	}
}

// HandleSetRole sets a new role
//
//	loginID is the ID or email of the user
//	newPassword can be empty to auto-generate a password
func HandleSetRole(co *consumer.Consumer, loginID string, newRole string) error {

	authnClient := authnpkg.NewAuthnAdminClient(co)
	prof, err := authnClient.GetClientProfile(loginID)
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
