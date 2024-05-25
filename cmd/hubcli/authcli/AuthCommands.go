package authcli

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/authnclient"
	"github.com/hiveot/hub/runtime/authz/authzclient"
	"golang.org/x/exp/rand"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

// AuthAddUserCommand adds a user
func AuthAddUserCommand(hc *hubclient.IHubClient) *cli.Command {
	displayName := ""
	role := ""
	rolesTxt := fmt.Sprintf("%s, %s, %s, %s",
		api.ClientRoleViewer, api.ClientRoleOperator, api.ClientRoleManager, api.ClientRoleAdmin)

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

// AuthAddServiceCommand adds a service with key and auth token
func AuthAddServiceCommand(hc *hubclient.IHubClient, certsDir *string) *cli.Command {
	displayName := ""

	return &cli.Command{
		Name:      "addsvc",
		Usage:     "Add a service with its key and auth token in the certs folder.",
		ArgsUsage: "<serviceID>",
		Category:  "auth",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "set a display name",
				Value:       displayName,
				Destination: &displayName,
			},
		},
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 1 {
				err := fmt.Errorf("expected 1 argument")
				return err
			}
			serviceID := cCtx.Args().First()
			err := HandleAddService(*hc, serviceID, displayName, *certsDir)
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

// AuthSetPasswordCommand sets a client's password
func AuthSetPasswordCommand(hc *hubclient.IHubClient) *cli.Command {
	return &cli.Command{
		Name:      "setpass",
		Usage:     "Set password. (careful, no confirmation)",
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

// HandleAddUser adds a user and displays a temporary password
func HandleAddUser(
	hc hubclient.IHubClient, loginID string, displayName string, role string) (err error) {

	newPassword := GeneratePassword(9, true)
	authnAdmin := authnclient.NewAuthnAdminClient(hc)
	authzAdmin := authzclient.NewAuthzClient(hc)

	err = authnAdmin.AddClient(api.ClientTypeUser, loginID, displayName, newPassword, "")
	prof, _ := authnAdmin.GetClientProfile(loginID)
	_ = authnAdmin.UpdateClientProfile(prof)
	_ = authzAdmin.SetClientRole(loginID, role)
	if err != nil {
		fmt.Println("Error: " + err.Error())
	} else if newPassword != "" {
		println("User " + loginID + " added successfully. Temp password: " + newPassword)
	} else {
		// no need to show the given password
		fmt.Println("User " + loginID + " added successfully")
	}
	return err
}

// HandleAddService adds a service with key and token
//
//	loginID is required
//	displayName is optional
//	certsDir with directory to store keys/token
func HandleAddService(
	hc hubclient.IHubClient, serviceID string, displayName string, certsDir string) (err error) {
	var kp keys.IHiveKey
	//TODO: use standardized extensions from launcher
	keyFile := serviceID + ".key"
	authnAdmin := authnclient.NewAuthnAdminClient(hc)

	// if a key exists, use it
	keyPath := path.Join(certsDir, keyFile)
	if _, err = os.Stat(keyPath); errors.Is(err, os.ErrNotExist) {
		kp = hc.CreateKeyPair()
		err = kp.ExportPrivateToFile(keyPath)
		pubKeyPath := path.Join(certsDir, serviceID+".pub")
		err = kp.ExportPublicToFile(pubKeyPath)
		fmt.Printf("New private/public keys written to file '%s'\n", keyPath)
	} else {
		kp = hc.CreateKeyPair()
		err = kp.ImportPrivateFromFile(keyPath)
		fmt.Printf("Private key loaded from file '%s'\n", keyPath)
	}
	if err != nil {
		slog.Error("Failed creating or loading key", "err", err.Error())
		return
	}
	err = authnAdmin.AddClient(
		api.ClientTypeService, serviceID, displayName, kp.ExportPrivate(), "")
	if err != nil {
		slog.Error("Failed adding service",
			"serviceID", serviceID, "err", err.Error())
		return
	} else {
		fmt.Printf("Service '%s' added succesfully\n", serviceID)
	}

	// service needs an auth token, remove existing
	tokenFile := serviceID + ".token"
	tokenPath := path.Join(certsDir, tokenFile)
	if _, err = os.Stat(tokenPath); errors.Is(err, os.ErrNotExist) {
		authToken, _ := authnAdmin.NewAuthToken(serviceID, 0)
		err = os.WriteFile(tokenPath, []byte(authToken), 0400)
		fmt.Printf("Auth token written to file '%s'\n", tokenPath)
	} else {
		fmt.Printf("Token file %s already exists. No changes made.\n", tokenPath)
	}

	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	return err
}

// HandleListClients shows a list of user profiles
func HandleListClients(hc hubclient.IHubClient) (err error) {

	authnAdmin := authnclient.NewAuthnAdminClient(hc)
	authzAdmin := authzclient.NewAuthzClient(hc)
	profileList, err := authnAdmin.GetProfiles()

	fmt.Println("Users")
	fmt.Println("Login ID             Display Name              Role            GetUpdated")
	fmt.Println("--------             ------------              ----            -------")
	for _, profile := range profileList {
		if profile.ClientType == api.ClientTypeUser {
			role, _ := authzAdmin.GetClientRole(profile.ClientID)
			fmt.Printf("%-20s %-25s %-15s %s\n",
				profile.ClientID,
				profile.DisplayName,
				role,
				utils.FormatMSE(profile.UpdatedMsec, false),
			)
		}
	}
	fmt.Println()
	fmt.Println("Devices/Services")
	fmt.Println("SenderID             Type            GetUpdated")
	fmt.Println("--------             ----            -------")
	for _, profile := range profileList {
		if profile.ClientType != api.ClientTypeUser {
			fmt.Printf("%-20s %-15s %s\n",
				profile.ClientID,
				profile.ClientType,
				utils.FormatMSE(profile.UpdatedMsec, false),
			)
		}
	}
	return err
}

// HandleRemoveClient removes a user
func HandleRemoveClient(hc hubclient.IHubClient, clientID string) (err error) {
	authnAdmin := authnclient.NewAuthnAdminClient(hc)
	err = authnAdmin.RemoveClient(clientID)

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
	authnAdmin := authnclient.NewAuthnAdminClient(hc)
	err := authnAdmin.SetClientPassword(loginID, newPassword)

	if err != nil {
		fmt.Println("Error: " + err.Error())
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
	authzAdmin := authzclient.NewAuthzClient(hc)

	err := authzAdmin.SetClientRole(loginID, newRole)

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
