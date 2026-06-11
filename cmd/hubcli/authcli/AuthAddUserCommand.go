package authcli

import (
	"crypto"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/hiveot/hivekit/go/modules/authn"
	authnpkg "github.com/hiveot/hivekit/go/modules/authn/pkg"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/urfave/cli/v2"
)

// AuthAddUserCommand adds a user
func AuthAddUserCommand(hc **consumer.Consumer) *cli.Command {
	displayName := ""
	var role string = string(authn.ClientRoleViewer)
	rolesTxt := fmt.Sprintf("[%s, %s, %s, %s]",
		authn.ClientRoleViewer, authn.ClientRoleOperator,
		authn.ClientRoleManager, authn.ClientRoleAdmin,
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
func AuthAddServiceCommand(hc **consumer.Consumer, certsDir *string) *cli.Command {
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
func AuthListClientsCommand(hc **consumer.Consumer) *cli.Command {
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
func AuthRemoveClientCommand(hc **consumer.Consumer) *cli.Command {
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
func AuthRoleCommand(hc **consumer.Consumer) *cli.Command {
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
	co *consumer.Consumer, loginID string, displayName string, role string) (err error) {

	newPassword := GeneratePassword(9, true)
	authnAdmin := authnpkg.NewAuthnAdminClient(co)

	token, err := authnAdmin.AddClient(loginID, displayName, authn.ClientRoleViewer, "")
	_ = token
	authnAdmin.SetClientPassword(loginID, newPassword)
	prof, _ := authnAdmin.GetClientProfile(loginID)
	_ = authnAdmin.UpdateClientProfile(prof)

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
	co *consumer.Consumer, serviceID string, displayName string, certsDir string) (err error) {

	var pubKey crypto.PublicKey
	var privKey crypto.PrivateKey
	//TODO: use standardized extensions from launcher
	keyFile := serviceID + ".key"

	// if a key exists, use it
	keyPath := path.Join(certsDir, keyFile)
	if _, err = os.Stat(keyPath); errors.Is(err, os.ErrNotExist) {
		privKey, pubKey = utils.NewEcdsaKey()
		err = utils.SavePrivateKey(privKey, keyPath) // ExportPrivateToFile(keyPath)
		pubKeyPath := path.Join(certsDir, serviceID+".pub")
		err = utils.SavePublicKey(pubKey, pubKeyPath)
		fmt.Printf("New private/public keys written to file '%s'\n", keyPath)
	} else {
		privKey, pubKey, err = utils.LoadPrivateKey(keyPath)
		fmt.Printf("Private key loaded from file '%s'\n", keyPath)
	}
	if err != nil {
		slog.Error("Failed creating or loading key", "err", err.Error())
		return
	}
	authAdmin := authnpkg.NewAuthnAdminClient(co)
	pubKeyPem := utils.PublicKeyToPem(pubKey)
	authToken, err := authAdmin.AddClient(serviceID, displayName, authn.ClientRoleService, pubKeyPem)
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
func HandleListClients(co *consumer.Consumer) (err error) {

	authnClient := authnpkg.NewAuthnAdminClient(co)
	authnClient.SetRequestSink(co)

	profileList, err := authnClient.GetProfiles()

	fmt.Println("Users")
	fmt.Println("Login ID             Display Name              Role            Modified")
	fmt.Println("--------             ------------              ----            -------")
	for _, profile := range profileList {
		if profile.Role != authn.ClientRoleAgent && profile.Role != authn.ClientRoleService {
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
		if profile.Role == authn.ClientRoleAgent || profile.Role == authn.ClientRoleService {
			fmt.Printf("%-20s %-15s %s\n",
				profile.ClientID,
				profile.Role,
				utils.FormatDateTime(profile.TimeUpdated, ""),
			)
		}
	}
	return err
}

// HandleRemoveClient removes a user
func HandleRemoveClient(co *consumer.Consumer, clientID string) (err error) {
	authnClient := authnpkg.NewAuthnAdminClient(co)
	authnClient.RemoveClient(clientID)

	if err != nil {
		fmt.Println("Error: " + err.Error())
	} else {
		fmt.Println("Client " + clientID + " removed")

	}
	return err
}
