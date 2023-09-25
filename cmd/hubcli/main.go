package main

import (
	"fmt"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/cmd/hubcli/corecli"
	"github.com/hiveot/hub/cmd/hubcli/launchercli"
	"github.com/hiveot/hub/lib/hubcl"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"github.com/urfave/cli/v2"
	"os"
	"path"
)

const Version = `0.1-alpha`

var binDir string
var homeDir string
var runDir string
var certsDir string
var configDir string
var nowrap bool

// CLI for managing the HiveOT Hub
//
// commandline:  hubcli command options

func main() {
	var hc hubclient.IHubClient
	var clientID = "admin"
	var verbose bool
	logging.SetLogging("warning", "")
	binDir = path.Dir(os.Args[0])
	homeDir = path.Dir(binDir)
	nowrap = false
	f := utils.GetFolders(homeDir, false)
	certsDir = f.Certs
	configDir = f.Config

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
				Value:       clientID,
				Destination: &clientID,
			},
			&cli.BoolFlag{
				Name:        "loginfo",
				Usage:       "verbose logging",
				Value:       verbose,
				Destination: &verbose,
			},
		},
		Before: func(c *cli.Context) (err error) {
			f = utils.GetFolders(homeDir, false)
			certsDir = f.Certs
			runDir = f.Run
			homeDir = f.Home
			configDir = f.Config
			//
			if verbose {
				logging.SetLogging("info", "")
			}
			if nowrap {
				fmt.Printf(utils.WrapOff)
			}
			hc, err = hubcl.ConnectToHub("", clientID, certsDir, "")
			return err
		},
		Commands: []*cli.Command{
			// pass paths by reference so they are updated in the Before section
			corecli.CreateCACommand(&certsDir),
			corecli.ViewCACommand(&certsDir),

			launchercli.LauncherListCommand(&hc),
			launchercli.LauncherStartCommand(&hc),
			launchercli.LauncherStopCommand(&hc),
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
