package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hiveot/hivekit/go/api"
	consumerrecipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/cmd/hubcli/authcli"
	"github.com/hiveot/hub/cmd/hubcli/certs"
	"github.com/hiveot/hub/cmd/hubcli/directorycli"
	"github.com/hiveot/hub/cmd/hubcli/historycli"
	"github.com/hiveot/hub/cmd/hubcli/launchercli"
	"github.com/hiveot/hub/cmd/hubcli/pubsubcli"
	"github.com/urfave/cli/v3"
)

const Version = `1.0-alpha`

// var env utils.AppEnvironment
var nowrap bool

// CLI for managing the HiveOT Hub
//
// commandline:  hubcli command options

func main() {
	// var co *consumer.Consumer
	var r *consumerrecipe.ConsumerRecipe
	var verbose bool
	var password = ""
	var homeDir string
	var certsDir string
	var serverURL string
	var authToken string

	// first obtain the environment defaults
	env := api.NewHiveEnvironment("", false)
	homeDir = env.HomeDir
	certsDir = env.CertsDir

	//defaultHome := env.HomeDir // to detect changes to the home directory
	utils.SetLogging("warning", "")
	nowrap = false

	// FIXME: use the flag from env?

	app := &cli.Command{
		EnableShellCompletion: true,
		Name:                  "hubcli",
		Usage:                 "Hub Commandline Interface",
		Version:               Version,

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "home",
				Usage:       "Path to application home directory",
				Value:       homeDir,
				Destination: &homeDir,
			},
			&cli.BoolFlag{
				Name:        "nowrap",
				Usage:       "Disable konsole wrapping",
				Value:       nowrap,
				Destination: &nowrap,
			},
			// &cli.StringFlag{
			// 	Name:        "login",
			// 	Usage:       "login ID",
			// 	Value:       loginID,
			// 	Destination: &loginID,
			// },
			&cli.StringFlag{
				Name:        "password",
				Usage:       "optional password for alt user",
				Value:       password,
				Destination: &password,
			},
			&cli.StringFlag{
				Name:        "serverURL",
				Usage:       "schema://addr:port/path (default: use DNS-SD discovery)",
				Value:       serverURL,
				Destination: &serverURL,
			},
			&cli.BoolFlag{
				Name:        "loginfo",
				Usage:       "verbose logging",
				Value:       verbose,
				Destination: &verbose,
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			// var cc api.ITransportClient
			var err error
			// reload env in case home changes
			env = api.NewHiveEnvironment(homeDir, false)
			certsDir = env.CertsDir
			if verbose {
				utils.SetLogging("info", "")
			}
			if nowrap {
				fmt.Printf(utils.WrapOff)
			}

			// most commands need auth
			authToken, _ = env.GetAuthToken()

			// TODO: cleanup: don't connect for these commands
			cmd := c.Args().First()
			if cmd == "" || cmd == "disco" || cmd == "cca" || cmd == "vca" {
				return ctx, nil
			}

			if authToken == "" && password == "" {
				return ctx, fmt.Errorf("hubcli: missing authentication token")
			}

			// Note that not all commands need a consumer connection, but create one anyways for simplicity.
			// The recipe has api for directory
			r, _, err = consumerrecipe.NewConsumerRecipe(env, false)
			if err != nil {
				os.Exit(1)
			}

			// caCert, _ := env.GetCACert()
			// if password != "" {
			// 	cc, err = clients.NewTransportClient("", serverURL, caCert)
			// 	if err == nil {
			// 		authncl := authn_client.NewUserAuthnHttpClient(serverURL, caCert)
			// 		authToken, err = authncl.LoginWithPassword(loginID, password)
			// 	}
			// 	cc, err = clients.NewTransportClient("", serverURL, caCert)
			// 	err = cc.SetAuthToken(loginID, authToken)
			// } else {
			// 	cc, err = clients.NewTransportClient("", serverURL, caCert)
			// 	err = cc.SetAuthToken(loginID, authToken)
			// }

			// if err != nil {
			// 	slog.Error("Unable to connect to the server", "err", err)
			// 	return ctx, fmt.Errorf("unable to connect to the hub")
			// }
			// co = consumer.NewConsumer(cc, nil)
			return ctx, nil
		},
		// FIXME: commands arguments must be passed by reference so because the
		// provided argument 'r' is instantiated in the Before section.
		Commands: []*cli.Command{
			// these commands work without a server connection
			certs.CreateCACommand(&certsDir),
			certs.ViewCACommand(&certsDir),

			authcli.AuthAddUserCommand(&r),
			authcli.AuthAddServiceCommand(&r, &env.CertsDir),
			authcli.AuthListClientsCommand(&r),
			authcli.AuthRemoveClientCommand(&r),
			authcli.AuthSetRoleCommand(&r),
			authcli.AuthSetPasswordCommand(&r),

			launchercli.LauncherListCommand(&r),
			launchercli.LauncherStartCommand(&r),
			launchercli.LauncherStopCommand(&r),

			directorycli.DirectoryListCommand(&r),
			directorycli.DiscoListCommand(&env),

			//historycli.HistoryLatestCommand(&hc),
			historycli.HistoryListCommand(&r),

			pubsubcli.PubActionCommand(&r),
			pubsubcli.SubEventsCommand(&r),
			pubsubcli.SubTDCommand(&r),
		},
	}

	// Show the arguments in the command line
	//	cli.AppHelpTemplate = `NAME:
	//  {{.ID}} - {{.Usage}}
	//USAGE:
	//  {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
	//  {{if len .Authors}}
	//AUTHOR:
	//  {{range .Authors}}{{ . }}{{end}}
	//  {{end}}{{if .Commands}}
	//COMMANDS: {{range .VisibleCategories}}{{if .ID}}
	//   {{.ID }}:{{"\t"}}{{range .VisibleCommands}}
	//      {{join .Names ", "}} {{.ArgsUsage}} {{"\t"}}{{.Usage}}{{end}}{{else}}{{template "visibleCommandTemplate" .}}{{end}}{{end}}
	//
	//GLOBAL OPTIONS:
	//  {{range .VisibleFlags}}{{.}}
	//  {{end}}
	//{{end}}
	//`
	app.Suggest = true
	app.HideHelpCommand = true
	if err := app.Run(context.Background(), os.Args); err != nil {
		println("ERROR: ", err.Error())
		//helpArgs := append(os.Args, "-h")
		//_ = app.Run(helpArgs)
	}
}
