package main

import (
	"fmt"
	"log/slog"
	"os"

	authnpkg "github.com/hiveot/hivekit/go/modules/authn/pkg"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/clients"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/cmd/hubcli/authcli"
	"github.com/hiveot/hub/cmd/hubcli/certs"
	"github.com/hiveot/hub/cmd/hubcli/directorycli"
	"github.com/hiveot/hub/cmd/hubcli/historycli"
	"github.com/hiveot/hub/cmd/hubcli/launchercli"
	"github.com/hiveot/hub/cmd/hubcli/pubsubcli"
	"github.com/urfave/cli/v2"
)

const Version = `1.0-alpha`

// var env utils.AppEnvironment
var nowrap bool

// CLI for managing the HiveOT Hub
//
// commandline:  hubcli command options

func main() {
	var co *consumer.Consumer
	var verbose bool
	var loginID = "admin"
	var password = ""
	var homeDir string
	var certsDir string
	var serverURL string
	var authToken string

	// environment defaults
	env := factory.NewAppEnvironment("", false)
	homeDir = env.HomeDir
	certsDir = env.CertsDir

	//defaultHome := env.HomeDir // to detect changes to the home directory
	utils.SetLogging("warning", "")
	nowrap = false

	app := &cli.App{
		EnableBashCompletion: true,
		Name:                 "hubcli",
		Usage:                "Hub Commandline Interface",
		Version:              Version,

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
			&cli.StringFlag{
				Name:        "login",
				Usage:       "login ID",
				Value:       loginID,
				Destination: &loginID,
			},
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
		Before: func(c *cli.Context) (err error) {
			var cc transport.ITransportClient
			// reload env in case home changes
			env = factory.NewAppEnvironment(homeDir, false)
			certsDir = env.CertsDir
			if verbose {
				utils.SetLogging("info", "")
			}
			if nowrap {
				fmt.Printf(utils.WrapOff)
			}

			// most commands need auth
			authToken = env.GetAppToken()

			// TODO: cleanup: don't connect for these commands
			cmd := c.Args().First()
			if cmd == "" || cmd == "disco" || cmd == "cca" || cmd == "vca" {
				return nil
			}

			if authToken == "" && password == "" {
				return fmt.Errorf("hubcli: missing authentication token")
			}
			caCert, _ := env.GetCA()
			if password != "" {
				cc, err = clients.NewTransportClient("", serverURL, caCert)
				if err == nil {
					authncl := authnpkg.NewUserAuthnHttpClient(serverURL, caCert)
					authToken, err = authncl.LoginWithPassword(loginID, password)
				}
				cc, err = clients.NewTransportClient("", serverURL, caCert)
				err = cc.AuthenticateWithToken(loginID, authToken)
			} else {
				cc, err = clients.NewTransportClient("", serverURL, caCert)
				err = cc.AuthenticateWithToken(loginID, authToken)
			}

			if err != nil {
				slog.Error("Unable to connect to the server", "err", err)
				return fmt.Errorf("unable to connect to the hub")
			}
			co = consumer.NewConsumer(nil)
			co.SetRequestSink(cc)
			cc.SetNotificationSink(co)
			return nil
		},
		// commands arguments are passed by reference so they are updated in the Before section
		Commands: []*cli.Command{
			// these commands work without a server connection
			certs.CreateCACommand(&certsDir),
			certs.ViewCACommand(&certsDir),

			authcli.AuthAddUserCommand(&co),
			authcli.AuthAddServiceCommand(&co, &env.CertsDir),
			authcli.AuthListClientsCommand(&co),
			authcli.AuthRemoveClientCommand(&co),
			authcli.AuthSetRoleCommand(&co),
			authcli.AuthSetPasswordCommand(&co),

			launchercli.LauncherListCommand(&co),
			launchercli.LauncherStartCommand(&co),
			launchercli.LauncherStopCommand(&co),

			directorycli.DirectoryListCommand(&co),
			directorycli.DiscoListCommand(loginID, &authToken),

			//historycli.HistoryLatestCommand(&hc),
			historycli.HistoryListCommand(&co),

			pubsubcli.PubActionCommand(&co),
			pubsubcli.SubEventsCommand(&co),
			pubsubcli.SubTDCommand(&co),
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
	if err := app.Run(os.Args); err != nil {
		println("ERROR: ", err.Error())
		//helpArgs := append(os.Args, "-h")
		//_ = app.Run(helpArgs)
	}
}
