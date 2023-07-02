package main

import (
	"fmt"
	"github.com/hiveot/hub/cmd/hubcli/corecli"
	"github.com/hiveot/hub/cmd/hubcli/launchercli"
	"github.com/hiveot/hub/lib/svcconfig"
	"github.com/hiveot/hub/lib/utils"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os"
	"path"
)

const Version = `0.1-alpha`

var binFolder string
var homeFolder string
var runFolder string
var certsFolder string
var configFolder string
var nowrap bool

// CLI for managing the HiveOT Hub
//
// commandline:  hubcli command options

func main() {
	//logging.SetLogging("info", "")
	binFolder = path.Dir(os.Args[0])
	homeFolder = path.Dir(binFolder)
	nowrap = false
	f := svcconfig.GetFolders(homeFolder, false)
	certsFolder = f.Certs
	configFolder = f.Config

	app := &cli.App{
		EnableBashCompletion: true,
		Name:                 "hubcli",
		Usage:                "Hub Commandline Interface",
		Version:              Version,

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "home",
				Usage:       "Path to home `folder`",
				Value:       homeFolder,
				Destination: &homeFolder,
			},
			&cli.BoolFlag{
				Name:        "nowrap",
				Usage:       "Disable konsole wrapping",
				Value:       nowrap,
				Destination: &nowrap,
			},
		},
		Before: func(c *cli.Context) error {
			f = svcconfig.GetFolders(homeFolder, false)
			certsFolder = f.Certs
			runFolder = f.Run
			configFolder = f.Config
			if nowrap {
				fmt.Printf(utils.WrapOff)
			}
			return nil
		},
		Commands: []*cli.Command{
			launchercli.LauncherListCommand(&runFolder),
			launchercli.LauncherStartCommand(&runFolder),
			launchercli.LauncherStopCommand(&runFolder),

			corecli.InitCommand(&certsFolder),
		},
	}

	// Show the arguments in the command line
	cli.AppHelpTemplate = `NAME:
  {{.Name}} - {{.Usage}}
USAGE:
  {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
  {{if len .Authors}}
AUTHOR:
  {{range .Authors}}{{ . }}{{end}}
  {{end}}{{if .Commands}}
COMMANDS: {{range .VisibleCategories}}{{if .Name}}
   {{.Name }}:{{"\t"}}{{range .VisibleCommands}}
      {{join .Names ", "}} {{.ArgsUsage}} {{"\t"}}{{.Usage}}{{end}}{{else}}{{template "visibleCommandTemplate" .}}{{end}}{{end}}

GLOBAL OPTIONS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
{{end}}
`
	app.Suggest = true
	app.HideHelpCommand = true
	if err := app.Run(os.Args); err != nil {
		logrus.Error("ERROR: ", err)
		helpArgs := append(os.Args, "-h")
		_ = app.Run(helpArgs)
	}
}
