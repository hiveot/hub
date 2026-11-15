package authcli

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/hiveot/hivekit/go/api"
	authn_client "github.com/hiveot/hivekit/go/cells/authn/client"
	certs_client "github.com/hiveot/hivekit/go/cells/certs/client"
	"github.com/hiveot/hivekit/go/cells/consumer"
	corecipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/urfave/cli/v3"
)

// AuthAddUserCommand adds a user
func AuthAddUserCommand(r **corecipe.ConsumerRecipe) *cli.Command {
	displayName := ""
	var role string = string(api.ClientRoleViewer)
	rolesTxt := fmt.Sprintf("[%s, %s, %s, %s]",
		api.ClientRoleViewer, api.ClientRoleOperator,
		api.ClientRoleManager, api.ClientRoleAdmin,
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
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 1 {
				err := fmt.Errorf("expected 1 argument")
				return err
			}
			loginID := cmd.Args().Get(0)
			err := HandleAddUser(*r, loginID, displayName, role)
			return err
		},
	}
}

// AuthAddServiceCommand adds a service with key and auth token
func AuthAddServiceCommand(r **corecipe.ConsumerRecipe, certsDir *string) *cli.Command {
	displayName := ""

	return &cli.Command{
		Name:      "addsvc",
		Usage:     "[-name displayName] Add a service with a client cert in the certs folder.",
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
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 1 {
				err := fmt.Errorf("expected 1 argument")
				return err
			}
			serviceID := cmd.Args().First()
			err := HandleAddService(*r, serviceID, displayName, *certsDir)
			return err
		},
	}
}

// AuthListClientsCommand lists user profiles
func AuthListClientsCommand(r **corecipe.ConsumerRecipe) *cli.Command {
	return &cli.Command{
		Name:     "lu",
		Usage:    "List users",
		Category: "auth",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				err := fmt.Errorf("too many arguments")
				return err
			}
			err := HandleListClients(*r)
			return err
		},
	}
}

// AuthRemoveClientCommand removes a user
func AuthRemoveClientCommand(r **corecipe.ConsumerRecipe) *cli.Command {
	return &cli.Command{
		Name:      "rm",
		Usage:     "Remove a user or service. (careful, no confirmation)",
		ArgsUsage: "<loginID>",
		Category:  "auth",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 1 {
				err := fmt.Errorf("expected 1 arguments")
				return err
			}
			loginID := cmd.Args().Get(0)
			err := HandleRemoveClient(*r, loginID)
			return err
		},
	}
}

// AuthRoleCommand changes a user's role
func AuthRoleCommand(r *consumer.IConsumer) *cli.Command {
	return &cli.Command{
		Name:      "setrole",
		Usage:     "Set a new role",
		ArgsUsage: "<clientID> <newrole>",
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

// HandleAddUser adds a user and displays a temporary password
func HandleAddUser(
	r consumer.IConsumer, loginID string, displayName string, role string) (err error) {

	newPassword := GeneratePassword(9, true)
	authnAdmin := authn_client.NewAuthnAdminClient("", r)

	token, err := authnAdmin.AddClient(loginID, displayName, api.ClientRoleViewer, "")
	_ = token
	authnAdmin.SetClientPassword(loginID, newPassword)
	prof, _ := authnAdmin.GetProfile(loginID)
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

// HandleAddService adds a service with client certificate
//
//	loginID is required
//	displayName is optional
//	certsDir with directory to store keys/token
func HandleAddService(
	co consumer.IConsumer, serviceID string, displayName string, certsDir string) (err error) {

	var privKey crypto.Signer

	// if a key exists, use it
	certPath := path.Join(certsDir, serviceID+api.DefaultCertFileSuffix)
	keyPath := path.Join(certsDir, serviceID+api.DefaultKeyFileSuffix)

	if _, err = os.Stat(keyPath); errors.Is(err, os.ErrNotExist) {
		privKey = utils.NewEd25519Key()
		err = utils.SavePrivateKey(privKey, keyPath) // ExportPrivateToFile(keyPath)
		fmt.Printf("New private key written to file '%s'\n", keyPath)
	} else {
		privKey, err = utils.LoadPrivateKey(keyPath)
		fmt.Printf("Private key loaded from file '%s'\n", keyPath)
	}
	if err != nil {
		slog.Error("Failed creating or loading key", "err", err.Error())
		return
	}
	authAdmin := authn_client.NewAuthnAdminClient("", co)
	pubKeyPem := utils.PublicKeyToPem(privKey.Public())
	authToken, err := authAdmin.AddClient(serviceID, displayName, api.ClientRoleService, pubKeyPem)
	_ = authToken
	if err != nil {
		slog.Error("Failed adding service",
			"serviceID", serviceID, "err", err.Error())
		return
	} else {
		fmt.Printf("Service '%s' added succesfully\n", serviceID)
	}

	// create/refresh a client certificate
	certsClient := certs_client.NewCertsClient("", co)
	x509cert, err := certsClient.CreateClientCert(serviceID, api.ClientRoleService, 0, privKey.Public())
	if err == nil {
		err = utils.SaveX509Cert(x509cert, certPath)
	}
	if err != nil {
		slog.Error("Failed creating Client cert",
			"serviceID", serviceID, "err", err.Error())
	} else {
		fmt.Printf("Client cert for '%s' created succesfully\n", serviceID)
	}

	return err
}

// HandleListClients shows a list of user profiles
func HandleListClients(co consumer.IConsumer) (err error) {

	authnClient := authn_client.NewAuthnAdminClient("", co)
	authnClient.SetRequestSink(co)

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

// HandleRemoveClient removes a user
func HandleRemoveClient(co consumer.IConsumer, clientID string) (err error) {
	authnClient := authn_client.NewAuthnAdminClient("", co)
	authnClient.RemoveClient(clientID)

	if err != nil {
		fmt.Println("Error: " + err.Error())
	} else {
		fmt.Println("Client " + clientID + " removed")

	}
	return err
}
