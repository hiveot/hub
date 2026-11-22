package main

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

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
	var r *consumerrecipe.ConsumerRecipe
	var env *api.AppEnvironment
	// var verbose bool
	var clientID string
	var certsDir string
	var dirTDURL string
	var homeDir string
	var logLevel string = "warn"
	var serverURL string
	var showEnv bool
	var timeout time.Duration = time.Second * 5

	// obtain the environment defaults
	userHome, _ := os.UserHomeDir()
	homeDir = filepath.Join(userHome, "bin", "hiveot")
	currentUser, _ := user.Current()
	clientID = currentUser.Username
	nowrap = false

	// FIXME: use the flag from env?

	app := &cli.Command{
		EnableShellCompletion: true,
		Name:                  "hubcli",
		Usage:                 "Hub Commandline Interface",
		Version:               Version,
		HideVersion:           true, // the -v flag is mine

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "home",
				Usage:       "Path to application home directory",
				Value:       homeDir,
				Destination: &homeDir,
			},
			&cli.StringFlag{
				Name:        "clientID",
				Usage:       "ClientID to authenticate with",
				Value:       clientID,
				Destination: &clientID,
			},
			&cli.StringFlag{
				Name:        "dirTD",
				Usage:       "URL where to download the directory TD (default: use DNS-SD discovery)",
				Value:       dirTDURL,
				Destination: &dirTDURL,
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
				Name:        "serverURL",
				Usage:       "schema://addr:port/path (default: use DNS-SD discovery)",
				Value:       serverURL,
				Destination: &serverURL,
			},
			&cli.BoolFlag{
				Name:        "env",
				Usage:       "Print environment info",
				Value:       showEnv,
				Destination: &showEnv,
			},
			&cli.StringFlag{
				Name:        "loglevel",
				Usage:       "Logging level debug|info|warn|error",
				Value:       logLevel,
				Destination: &logLevel,
			},
			&cli.DurationFlag{
				Name:        "timeout",
				Usage:       "RPC timeout for testing",
				Value:       timeout,
				Destination: &timeout,
			},
			// &cli.BoolFlag{
			// 	Name:        "v",
			// 	Usage:       "verbose logging",
			// 	Value:       verbose,
			// 	Destination: &verbose,
			// },
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			var err error
			// load env after 'home' is set
			env = api.NewAppEnvironment(homeDir, false)
			env.ClientID = clientID
			env.DirTDURL = dirTDURL
			env.LogLevel = logLevel
			env.ServerURL = serverURL
			env.SetRpcTimeout(timeout)
			utils.SetLogging(env.LogLevel, "")
			certsDir = env.CertsDir // some commands need a certs dir
			// if verbose {
			// utils.SetLogging("info", "")
			// }
			if showEnv {
				env.Print(2)
			}

			if nowrap {
				fmt.Printf(utils.WrapOff)
			}

			// These commands don't need a connection.
			cmd := c.Args().First()
			if cmd == "" || cmd == "disco" || cmd == "cca" || cmd == "vca" {
				return ctx, nil
			}

			// The recipe has api for directory
			r, _, err = consumerrecipe.NewConsumerRecipe(env, false)
			if err != nil {
				os.Exit(1)
			}

			return ctx, nil
		},
		// FIXME: commands arguments must be passed by reference because the provided argument
		// is instantiated in the Before section above, instead of when creating the commands.
		Commands: []*cli.Command{
			// these commands work without a server connection
			certs.ViewCertsCommand(&env),
			// certs.CreateCACommand(&env),

			authcli.AuthAddUserCommand(&r),
			authcli.AuthAddServiceCommand(&r, &certsDir),
			authcli.AuthListClientsCommand(&r),
			authcli.AuthRemoveClientCommand(&r),
			authcli.AuthSetRoleCommand(&r),
			authcli.AuthSetPasswordCommand(&r),
			authcli.AuthLoginCommand(&r),

			launchercli.LauncherListCommand(&r),
			launchercli.LauncherStartCommand(&r),
			launchercli.LauncherStopCommand(&r),

			directorycli.DirectoryListCommand(&r),
			directorycli.DiscoListCommand(&env),

			//historycli.HistoryLatestCommand(&hc),
			historycli.HistoryListCommand(&r),

			pubsubcli.PubActionCommand(&r),
			pubsubcli.SubCommand(&r),
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
