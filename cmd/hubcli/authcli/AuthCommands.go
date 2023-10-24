package authcli

import (
	"fmt"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"golang.org/x/exp/rand"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

// AuthnAddUserCommand adds a user
func AuthAddUserCommand(hc *hubclient.IHubClient) *cli.Command {
	displayName := ""
	role := ""
	rolesTxt := fmt.Sprintf("%s, %s, %s, %s",
		authapi.ClientRoleViewer, authapi.ClientRoleOperator, authapi.ClientRoleManager, authapi.ClientRoleAdmin)

	return &cli.Command{
		Name:      "addu",
		Usage:     "Add a user with role and generate a temporary password",
		ArgsUsage: "<userID> <role>",
		Category:  "auth",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "role",
				Usage:       rolesTxt,
				Value:       role,
				Destination: &role,
			},
			&cli.StringFlag{
				Name:        "name",
				Usage:       "",
				Value:       displayName,
				Destination: &displayName,
			},
		},
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() < 1 {
				err := fmt.Errorf("expected 1 or 2 arguments")
				return err
			}
			if cCtx.NArg() == 2 {
				role = cCtx.Args().Get(1)
			}
			loginID := cCtx.Args().Get(0)
			err := HandleAddUser(*hc, loginID, displayName, role)
			return err
		},
	}
}

// AuthListClientsCommand lists user profiles
func AuthListClientsCommand(hc *hubclient.IHubClient) *cli.Command {
	return &cli.Command{
		Name:     "lu",
		Usage:    "List users",
		Category: "auth",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() > 0 {
				err := fmt.Errorf("too many arguments")
				return err
			}
			err := HandleListClients(*hc)
			return err
		},
	}
}

// AuthRemoveClientCommand removes a user
func AuthRemoveClientCommand(hc *hubclient.IHubClient) *cli.Command {
	return &cli.Command{
		Name:      "rmu",
		Usage:     "Remove a user. (careful, no confirmation)",
		ArgsUsage: "<loginID>",
		Category:  "auth",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 1 {
				err := fmt.Errorf("expected 1 arguments")
				return err
			}
			loginID := cCtx.Args().Get(0)
			err := HandleRemoveClient(*hc, loginID)
			return err
		},
	}
}

// AuthPasswordCommand replaces a user's password
func AuthPasswordCommand(hc *hubclient.IHubClient) *cli.Command {
	return &cli.Command{
		Name:      "password",
		Usage:     "Change password. (careful, no confirmation)",
		ArgsUsage: "<loginID> <newpass>",
		Category:  "auth",
		Action: func(cCtx *cli.Context) error {
			newPassword := ""
			if cCtx.NArg() != 2 {
				err := fmt.Errorf("expected 2 arguments")
				return err
			}
			loginID := cCtx.Args().Get(0)
			newPassword = cCtx.Args().Get(1)
			err := HandleSetPassword(*hc, loginID, newPassword)
			return err
		},
	}
}

// AuthRoleCommand changes a user's role
func AuthRoleCommand(hc *hubclient.IHubClient) *cli.Command {
	return &cli.Command{
		Name:      "setrole",
		Usage:     "Set a new role",
		ArgsUsage: "<loginID> <newrole>",
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

// HandleAddUser adds a user and displays a temperary password
func HandleAddUser(
	hc hubclient.IHubClient, loginID string, displayName string, role string) (err error) {

	newPassword := GeneratePassword(9, true)
	authn := authclient.NewManageClients(hc)

	_, err = authn.AddUser(loginID, displayName, newPassword, "", role)

	if err != nil {
		fmt.Println("Error: " + err.Error())
	} else if newPassword == "" {
		fmt.Println("User " + loginID + " added successfully. Temp password: " + newPassword)
	} else {
		// no need to show the given password
		fmt.Println("User " + loginID + " added successfully")
	}
	return err
}

// HandleListClients shows a list of user profiles
func HandleListClients(hc hubclient.IHubClient) (err error) {

	authn := authclient.NewManageClients(hc)
	profileList, err := authn.GetProfiles()

	fmt.Println("Users")
	fmt.Println("Login ID             Display Name              Role            Updated")
	fmt.Println("--------             ------------              ----            -------")
	for _, profile := range profileList {
		if profile.ClientType == authapi.ClientTypeUser {
			fmt.Printf("%-20s %-25s %-15s %s\n",
				profile.ClientID,
				profile.DisplayName,
				profile.Role,
				utils.FormatMSE(profile.UpdatedMSE, false),
			)
		}
	}
	fmt.Println()
	fmt.Println("Devices/Services")
	fmt.Println("ClientID             Type            Updated")
	fmt.Println("--------             ----            -------")
	for _, profile := range profileList {
		if profile.ClientType != authapi.ClientTypeUser {
			fmt.Printf("%-20s %-15s %s\n",
				profile.ClientID,
				profile.ClientType,
				utils.FormatMSE(profile.UpdatedMSE, false),
			)
		}
	}
	return err
}

// HandleRemoveClient removes a user
func HandleRemoveClient(hc hubclient.IHubClient, clientID string) (err error) {
	authn := authclient.NewManageClients(hc)
	err = authn.RemoveClient(clientID)

	if err != nil {
		fmt.Println("Error: " + err.Error())
	} else {
		fmt.Println("Client " + clientID + " removed")

	}
	return err
}

// HandleSetPassword resets or replaces a password
//
//	loginID is the ID or email of the user
//	newPassword can be empty to auto-generate a password
func HandleSetPassword(hc hubclient.IHubClient, loginID string, newPassword string) error {
	if newPassword == "" {
		newPassword = GeneratePassword(9, true)
	}
	authn := authclient.NewManageClients(hc)
	err := authn.UpdateClientPassword(loginID, newPassword)

	if err != nil {
		fmt.Println("Error: " + err.Error())
	} else if newPassword == "" {
		fmt.Println("User "+loginID+" password has been updated. Generated password:", newPassword)
	} else {
		fmt.Println("User " + loginID + " password has been updated")
	}
	return err
}

// HandleSetRole sets a new role
//
//	loginID is the ID or email of the user
//	newPassword can be empty to auto-generate a password
func HandleSetRole(hc hubclient.IHubClient, loginID string, newRole string) error {
	authn := authclient.NewManageClients(hc)
	prof, err := authn.GetProfile(loginID)
	if err == nil {
		prof.Role = newRole
		err = authn.UpdateClient(loginID, prof)
	}

	if err != nil {
		fmt.Println("Error: " + err.Error())
	} else {
		fmt.Println("User " + loginID + " role has been updated to " + newRole)
	}
	return err
}

// GeneratePassword with upper, lower, numbers and special characters
func GeneratePassword(length int, useSpecial bool) (password string) {
	const charsLow = "abcdefghijklmnopqrstuvwxyz"
	const charsUpper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const charsSpecial = "!#$%&*+-./:=?@^_"
	const numbers = "0123456789"
	var pool = []rune(charsLow + numbers + charsUpper)

	if length < 2 {
		length = 8
	}
	if useSpecial {
		pool = append(pool, []rune(charsSpecial)...)
	}
	rand.Seed(uint64(time.Now().Unix()))
	//pwchars := make([]string, length)
	pwchars := strings.Builder{}

	for i := 0; i < length; i++ {
		pos := rand.Intn(len(pool))
		pwchars.WriteRune(pool[pos])
	}
	password = pwchars.String()
	return password
}
