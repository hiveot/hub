package main

import (
	"fmt"
	corecli2 "github.com/hiveot/hub/cmd/hubcli/corecli"
	"github.com/hiveot/hub/cmd/hubcli/launchercli"
	"github.com/hiveot/hub/lib/svcconfig"
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
	//logging.SetLogging("info", "")
	binDir = path.Dir(os.Args[0])
	homeDir = path.Dir(binDir)
	nowrap = false
	f := svcconfig.GetFolders(homeDir, false)
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
				Usage:       "Path to home `folder`",
				Value:       homeDir,
				Destination: &homeDir,
			},
			&cli.BoolFlag{
				Name:        "nowrap",
				Usage:       "Disable konsole wrapping",
				Value:       nowrap,
				Destination: &nowrap,
			},
		},
		Before: func(c *cli.Context) error {
			f = svcconfig.GetFolders(homeDir, false)
			certsDir = f.Certs
			runDir = f.Run
			homeDir = f.Home
			configDir = f.Config
			if nowrap {
				fmt.Printf(utils.WrapOff)
			}
			return nil
		},
		Commands: []*cli.Command{
			// pass paths by reference so they are updated in the Before section
			corecli2.CreateCACommand(&certsDir),
			corecli2.ViewCACommand(&certsDir),
			corecli2.RunCommand(&certsDir),
			corecli2.SetupCommand(&homeDir),

			launchercli.LauncherListCommand(&runDir),
			launchercli.LauncherStartCommand(&runDir),
			launchercli.LauncherStopCommand(&runDir),
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
