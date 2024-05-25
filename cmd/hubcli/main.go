package main

import (
	"fmt"
	"github.com/hiveot/hub/cmd/hubcli/authcli"
	"github.com/hiveot/hub/cmd/hubcli/certs"
	"github.com/hiveot/hub/cmd/hubcli/directorycli"
	"github.com/hiveot/hub/cmd/hubcli/historycli"
	"github.com/hiveot/hub/cmd/hubcli/idprovcli"
	"github.com/hiveot/hub/cmd/hubcli/launchercli"
	"github.com/hiveot/hub/cmd/hubcli/pubsubcli"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/connect"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/lib/utils"
	"github.com/urfave/cli/v2"
	"log/slog"
	"os"
)

const Version = `0.1-alpha`

// var env utils.AppEnvironment
var nowrap bool

// CLI for managing the HiveOT Hub
//
// commandline:  hubcli command options

func main() {
	var hc hubclient.IHubClient
	var verbose bool
	var loginID = "admin"
	var password = ""
	var homeDir string
	var certsDir string
	var serverURL string

	// environment defaults
	env := plugin.GetAppEnvironment("", false)
	homeDir = env.HomeDir
	certsDir = env.CertsDir

	//defaultHome := env.HomeDir // to detect changes to the home directory
	logging.SetLogging("warning", "")
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
				Name:        "server",
				Usage:       "server URL (default: use DNS-SD discovery)",
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
			// reload env in case home changes
			env = plugin.GetAppEnvironment(homeDir, false)
			certsDir = env.CertsDir
			if verbose {
				logging.SetLogging("info", "")
			}
			if nowrap {
				fmt.Printf(utils.WrapOff)
			}
			hc, err = connect.ConnectToHub(serverURL, loginID, certsDir, password)
			if err != nil {
				slog.Warn("Unable to connect to the server", "err", err)
			}
			return nil
		},
		// commands arguments are passed by reference so they are updated in the Before section
		Commands: []*cli.Command{
			// these commands work without a server connection
			certs.CreateCACommand(&certsDir),
			certs.ViewCACommand(&certsDir),

			authcli.AuthAddUserCommand(&hc),
			authcli.AuthAddServiceCommand(&hc, &env.CertsDir),
			authcli.AuthListClientsCommand(&hc),
			authcli.AuthRemoveClientCommand(&hc),
			authcli.AuthSetPasswordCommand(&hc),

			launchercli.LauncherListCommand(&hc),
			launchercli.LauncherStartCommand(&hc),
			launchercli.LauncherStopCommand(&hc),

			directorycli.DirectoryListCommand(&hc),

			//historycli.HistoryLatestCommand(&hc),
			historycli.HistoryListCommand(&hc),

			pubsubcli.PubActionCommand(&hc),
			pubsubcli.SubEventsCommand(&hc),
			pubsubcli.SubTDCommand(&hc),

			idprovcli.ProvisionListCommand(&hc),
			idprovcli.ProvisionRequestCommand(&hc),
			idprovcli.ProvisionApproveRequestCommand(&hc),
			idprovcli.ProvisionPreApproveCommand(&hc),
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
