package corecli

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"os/exec"
	"path"
)

// SetupCommand setup the runtime environment folders and keys
// This does not require any services to run.
//
//	hubcli setup [--new]
func SetupCommand(homeDir *string) *cli.Command {
	var newSetup bool

	return &cli.Command{
		Name:     "setup",
		Category: "core",
		Usage:    "Create missing folders and keys",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "new",
				Usage:       "Create a new environment. Use with care!",
				Value:       newSetup,
				Destination: &newSetup,
			},
		},
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() > 0 {
				return fmt.Errorf("unexpected argument(s) '%s'", cCtx.Args().First())
			}
			err := HandleSetup(*homeDir, newSetup)
			return err
		},
	}
}

func HandleSetup(homeDir string, newSetup bool) error {
	runCmd := path.Join(homeDir, "bin", "natscore")
	cmd := exec.Command(runCmd, "--home", homeDir, "setup")

	out, err := cmd.CombinedOutput()
	//err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("failed command: %s: %w", runCmd, err)
		fmt.Println(string(out))
	} else {
		fmt.Println("command success:", string(out))
	}
	return err
}
