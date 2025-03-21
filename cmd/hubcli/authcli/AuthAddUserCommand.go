package authcli

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging"
	authn "github.com/hiveot/hub/runtime/authn/api"
	authz "github.com/hiveot/hub/runtime/authz/api"
	"github.com/urfave/cli/v2"
	"log/slog"
	"os"
	"path"
)

// AuthAddUserCommand adds a user
func AuthAddUserCommand(hc **messaging.Consumer) *cli.Command {
	displayName := ""
	var role string = string(authz.ClientRoleViewer)
	rolesTxt := fmt.Sprintf("[%s, %s, %s, %s]",
		authz.ClientRoleViewer, authz.ClientRoleOperator,
		authz.ClientRoleManager, authz.ClientRoleAdmin,
	)

	return &cli.Command{
		Name:      "addu",
		Usage:     "Add a user with role and generate a temporary password",
		ArgsUsage: "<userID>",
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
			if cCtx.NArg() != 1 {
				err := fmt.Errorf("expected 1 argument")
				return err
			}
			loginID := cCtx.Args().Get(0)
			err := HandleAddUser(*hc, loginID, displayName, role)
			return err
		},
	}
}

// AuthAddServiceCommand adds a service with key and auth token
func AuthAddServiceCommand(hc **messaging.Consumer, certsDir *string) *cli.Command {
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
func AuthListClientsCommand(hc **messaging.Consumer) *cli.Command {
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
func AuthRemoveClientCommand(hc **messaging.Consumer) *cli.Command {
	return &cli.Command{
		Name:      "rm",
		Usage:     "Remove a user or service. (careful, no confirmation)",
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

// AuthRoleCommand changes a user's role
func AuthRoleCommand(hc **messaging.Consumer) *cli.Command {
	return &cli.Command{
		Name:      "setrole",
		Usage:     "Set a new role",
		ArgsUsage: "<clientID> <newrole>",
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
	hc *messaging.Consumer, loginID string, displayName string, role string) (err error) {

	newPassword := GeneratePassword(9, true)

	err = authn.AdminAddConsumer(hc, loginID, displayName, newPassword)
	prof, _ := authn.AdminGetClientProfile(hc, loginID)
	_ = authn.AdminUpdateClientProfile(hc, prof)
	_ = authz.AdminSetClientRole(hc, loginID, authz.ClientRole(role))
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
	hc *messaging.Consumer, serviceID string, displayName string, certsDir string) (err error) {
	var kp keys.IHiveKey
	//TODO: use standardized extensions from launcher
	keyFile := serviceID + ".key"

	// if a key exists, use it
	keyPath := path.Join(certsDir, keyFile)
	if _, err = os.Stat(keyPath); errors.Is(err, os.ErrNotExist) {
		kp = keys.NewEcdsaKey()
		err = kp.ExportPrivateToFile(keyPath)
		pubKeyPath := path.Join(certsDir, serviceID+".pub")
		err = kp.ExportPublicToFile(pubKeyPath)
		fmt.Printf("New private/public keys written to file '%s'\n", keyPath)
	} else {
		kp = keys.NewEcdsaKey()
		err = kp.ImportPrivateFromFile(keyPath)
		fmt.Printf("Private key loaded from file '%s'\n", keyPath)
	}
	if err != nil {
		slog.Error("Failed creating or loading key", "err", err.Error())
		return
	}
	authToken, err := authn.AdminAddService(hc, serviceID, displayName, kp.ExportPrivate())
	_ = authToken
	if err != nil {
		slog.Error("Failed adding service",
			"serviceID", serviceID, "err", err.Error())
		return
	} else {
		fmt.Printf("Service '%s' added succesfully\n", serviceID)
	}

	// service needs an auth token, remove existing
	//tokenFile := serviceID + ".token"
	//tokenPath := path.Join(certsDir, tokenFile)
	//if _, err = os.Stat(tokenPath); errors.Is(err, os.ErrNotExist) {
	//	authToken, _ := authnAdmin.NewAgentToken(serviceID, 0)
	//	err = os.WriteFile(tokenPath, []byte(authToken), 0400)
	//	fmt.Printf("Auth token written to file '%s'\n", tokenPath)
	//} else {
	//	fmt.Printf("Token file %s already exists. No changes made.\n", tokenPath)
	//}

	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	return err
}

// HandleListClients shows a list of user profiles
func HandleListClients(hc *messaging.Consumer) (err error) {

	profileList, err := authn.AdminGetProfiles(hc)

	fmt.Println("Users")
	fmt.Println("Login ID             Display Name              Role            Modified")
	fmt.Println("--------             ------------              ----            -------")
	for _, profile := range profileList {
		if profile.ClientType == authn.ClientTypeConsumer {
			role, _ := authz.AdminGetClientRole(hc, profile.ClientID)
			fmt.Printf("%-20s %-25s %-15s %s\n",
				profile.ClientID,
				profile.DisplayName,
				role,
				utils.FormatMSE(profile.Updated, false),
			)
		}
	}
	fmt.Println()
	fmt.Println("Devices/Services")
	fmt.Println("SenderID             Type            Modified")
	fmt.Println("--------             ----            -------")
	for _, profile := range profileList {
		if profile.ClientType != authn.ClientTypeConsumer {
			fmt.Printf("%-20s %-15s %s\n",
				profile.ClientID,
				profile.ClientType,
				utils.FormatMSE(profile.Updated, false),
			)
		}
	}
	return err
}

// HandleRemoveClient removes a user
func HandleRemoveClient(hc *messaging.Consumer, clientID string) (err error) {
	err = authn.AdminRemoveClient(hc, clientID)

	if err != nil {
		fmt.Println("Error: " + err.Error())
	} else {
		fmt.Println("Client " + clientID + " removed")

	}
	return err
}
