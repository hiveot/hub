package authcli

import (
	"context"
	"fmt"
	"os"

	"github.com/hiveot/hivekit/go/api"
	authn_client "github.com/hiveot/hivekit/go/cells/authn/client"
	consumer_recipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

// AuthLoginCommand login and obtain an authentication token
func AuthLoginCommand(r **consumer_recipe.ConsumerRecipe) *cli.Command {

	return &cli.Command{
		Name:      "login",
		Usage:     "Login and enter a password when prompted",
		ArgsUsage: "<loginID>",
		UsageText: "",
		Category:  "auth",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 1 {
				err := fmt.Errorf("expected 1 arguments")
				return err
			}
			loginID := cmd.Args().Get(0)
			err := HandleLogin(*r, loginID)
			return err
		},
	}
}

// HandlLogin logs is as a user and obtain a session token
//
//	loginID is the ID or email of the user
func HandleLogin(r *consumer_recipe.ConsumerRecipe, loginID string) error {

	env := r.GetEnv()
	rootCAs := env.GetRootCAs()
	serverURL := env.ServerURL

	// the serverURL comes from discovery
	// authnThingID := authn.AuthnUserServiceCellType
	if serverURL == "" {
		dirTD := env.GetDirTD()
		if dirTD == nil {
			return fmt.Errorf("Can't locate server or directory. Please provide option --serverURL <url>")
		}
		form, _ := dirTD.GetForm("", "", api.WotHttpBasicScheme, "")
		if form != nil {
			serverURL, _ = form.GetHRef()
		}
	}
	if serverURL == "" {
		return fmt.Errorf("Can't locate server for login. Please provide option --serverURL <url>")
	}

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return fmt.Errorf("not an input terminal")
	}
	fmt.Print("Password for " + loginID + ":")
	passwd, err := term.ReadPassword(fd)

	fmt.Println()

	authnClient := authn_client.NewUserAuthnHttpClient(serverURL, rootCAs)
	token, err := authnClient.LoginWithPassword(loginID, string(passwd))
	if err == nil {
		err = authnClient.SaveToken(env.CertsDir, loginID, token)
	}

	if err != nil {
		fmt.Println("Error: " + err.Error())
	} else {
		fmt.Println("Login successful for user " + loginID)
	}
	return err
}
