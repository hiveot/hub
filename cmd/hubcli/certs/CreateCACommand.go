package certs

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/hiveot/hivekit/go/cells/certs"
	certspkg "github.com/hiveot/hivekit/go/cells/certs/pkg"
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
	var validityDays = 365 * 5

	return &cli.Command{
		Name:     "cca",
		Usage:    "Create a new Hub CA certificate",
		Category: "certs",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "days",
				Usage:       "Number of `days` the certificate is valid.",
				Value:       validityDays,
				Destination: &validityDays,
			},
			&cli.BoolFlag{
				Name:        "force",
				Usage:       "Force overwrites an existing certificate and key.",
				Destination: &force,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				return fmt.Errorf("unexpected argument(s) '%s'", cmd.Args().First())
			}
			err := HandleCreateCACert(*certsFolder, cmd.Int("days"), cmd.Bool("force"))
			return err
		},
	}
}

// HandleCreateCACert generates the hub self-signed CA private key and certificate
// in the given folder.
// Use force to create the folder and overwrite existing certificate if it exists
func HandleCreateCACert(certsFolder string, validityDays int, force bool) error {
	caCertPath := path.Join(certsFolder, certs.DefaultCaCertFile)
	caKeyPath := path.Join(certsFolder, certs.DefaultCaKeyFile)

	// folder doesn't exist
	if _, err := os.Stat(certsFolder); err != nil {
		if force {
			_ = os.Mkdir(certsFolder, 0744)
		} else {
			return fmt.Errorf("certificate folder '%s' doesn't exist", certsFolder)
		}
	}
	// do not overwrite existing certificate unless force is used
	if !force {
		if _, err := os.Stat(caCertPath); err == nil {
			return fmt.Errorf("CA certificate already exists in '%s'. Use --force to replace", caCertPath)
		}
		if _, err := os.Stat(caKeyPath); err == nil {
			return fmt.Errorf("CA key alread exists in '%s'", caKeyPath)
		}
	}
	// the certsservice creates the CA cert and key on start
	svc := certspkg.NewCertsService(certsFolder)
	err := svc.Start()
	// caCert, caKey, err := svc.CreateCACert("Hub CA", validityDays)
	// if err != nil {
	// 	return err
	// }
	// err = certutils.SaveX509CertToPEM(caCert, caCertPath)
	// if err == nil {
	// 	err = caKey.ExportPrivateToFile(caKeyPath)
	// }

	slog.Info("Generated CA certificate", "caCertPath", caCertPath, "caKeyPath", caKeyPath)
	return err
}
