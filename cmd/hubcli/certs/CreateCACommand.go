package certs

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/certs"
	certs_service "github.com/hiveot/hivekit/go/cells/certs/service"
	"github.com/urfave/cli/v3"
)

// CreateCACommand create the Hub self-signed CA, valid for 10 years
// This does not require any services to run.
// To replace an existing CA, use the --force option
// After creating a new CA, services have to be restarted.
//
//	hubcli newca [--certs=CertFolder]  [--force]
func CreateCACommand(env **api.AppEnvironment) *cli.Command {
	var force = false
	// var validityDays = 365 * 5

	return &cli.Command{
		Name:     "cca",
		Usage:    "Create a new Hub CA certificate",
		Category: "certs",
		Flags: []cli.Flag{
			// &cli.IntFlag{
			// 	Name:        "days",
			// 	Usage:       "Number of `days` the certificate is valid.",
			// 	Value:       validityDays,
			// 	Destination: &validityDays,
			// },
			&cli.BoolFlag{
				Name:        "force",
				Usage:       "Force replacing an existing certificate and key.",
				Destination: &force,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				return fmt.Errorf("unexpected argument(s) '%s'", cmd.Args().First())
			}
			err := HandleCreateCACert(*env, cmd.Bool("force"))
			return err
		},
	}
}

// HandleCreateCACert refreshes the hub self-signed CA private key and certificate in
// the app environment certs directory.
// If force is set then force replacing the existing certificate.
func HandleCreateCACert(env *api.AppEnvironment, force bool) error {

	// Deleting a CA means all other certs become invalid
	if force {
		caCertPath := path.Join(env.CertsDir, api.DefaultCaCertFile)
		caKeyPath := path.Join(env.CertsDir, api.DefaultCaKeyFile)
		os.RemoveAll(caCertPath)
		os.RemoveAll(caKeyPath)
		fmt.Printf("Deleting CA certificate and key in '%s'", env.CertsDir)
	}
	// the certsservice creates the CA cert and key on start
	cfg := certs.CertsConfig{}
	svc, err := certs_service.NewCertsService(env, &cfg)
	_ = svc
	if err != nil {
		return err
	} else {
		svc.Stop()
	}

	return err
}
