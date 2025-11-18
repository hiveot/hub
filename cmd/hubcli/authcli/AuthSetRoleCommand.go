package authcli

import (
	"fmt"
	"strings"

	authz "github.com/hiveot/hivehub/runtime/authz/api"
	"github.com/hiveot/hivekitgo/messaging"
	"github.com/urfave/cli/v2"
)

// AuthSetRoleCommand changes a user's role
func AuthSetRoleCommand(hc **messaging.Consumer) *cli.Command {
	validRoles := []string{
		string(authz.ClientRoleViewer), string(authz.ClientRoleOperator),
		string(authz.ClientRoleManager), string(authz.ClientRoleAdmin),
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
func HandleSetRole(hc *messaging.Consumer, loginID string, newRole string) error {

	err := authz.AdminSetClientRole(hc, loginID, authz.ClientRole(newRole))

	if err != nil {
		//fmt.Println("Error: " + err.Error())
	} else {
		fmt.Println("User " + loginID + " role has been updated to " + newRole)
	}
	return err
}
