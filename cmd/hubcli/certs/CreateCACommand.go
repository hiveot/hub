package certs

import (
	"context"
	"fmt"
	"log/slog"
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
func CreateCACommand(certsFolder *string) *cli.Command {
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
			err := HandleCreateCACert(*certsFolder, cmd.Bool("force"))
			return err
		},
	}
}

// HandleCreateCACert generates the hub self-signed CA private key and certificate
// in the given folder.
// If force is set then force replacing the existing certificate
func HandleCreateCACert(certsDir string, force bool) error {
	caCertPath := path.Join(certsDir, api.DefaultCaCertFile)
	caKeyPath := path.Join(certsDir, api.DefaultCaKeyFile)

	// folder doesn't exist
	if _, err := os.Stat(certsDir); err != nil {
		fmt.Printf("Certificate folder '%s' doesn't exist. Creating it.\n", certsDir)
		err = os.Mkdir(certsDir, 0744)
		if err != nil {
			return fmt.Errorf("Failed creating certificate folder '%s' doesn't exist", certsDir)
		}
	}
	// replace existing certificate if force is used
	if force {
		os.Remove(caCertPath)
		os.Remove(caKeyPath)
		if _, err := os.Stat(caCertPath); err == nil {
			fmt.Printf("Deleting CA certificate in '%s'", caCertPath)
			err = os.Remove(caCertPath)
		}
		if _, err := os.Stat(caKeyPath); err == nil {
			fmt.Printf("Deleting CA key in '%s'", caKeyPath)
			err = os.Remove(caKeyPath)
		}
	}
	// the certsservice creates the CA cert and key on start
	cfg := certs.CertsConfig{}
	svc, err := certs_service.NewCertsService(certsDir, &cfg)
	_ = svc
	if err != nil {
		return err
	}

	slog.Info("Generated CA certificate", "caCertPath", caCertPath, "caKeyPath", caKeyPath)
	return err
}
