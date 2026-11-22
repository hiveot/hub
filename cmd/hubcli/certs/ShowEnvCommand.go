package certs

import (
	"context"

	"github.com/hiveot/hivekit/go/api"
	"github.com/urfave/cli/v3"
)

// ShowEnvCommand prints the application environment with certificates
func ShowEnvCommand(env **api.AppEnvironment) *cli.Command {

	return &cli.Command{
		Name:     "env",
		Usage:    "Show application environment and certificates",
		Category: "certs",
		Flags:    []cli.Flag{},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			(*env).Print(2)
			return nil
		},
	}
}
