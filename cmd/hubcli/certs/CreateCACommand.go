package certs

import (
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/hiveot/gocore/certs"
	"github.com/urfave/cli/v2"
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
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() > 0 {
				return fmt.Errorf("unexpected argument(s) '%s'", cCtx.Args().First())
			}
			err := HandleCreateCACert(*certsFolder, cCtx.Int("days"), cCtx.Bool("force"))
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

	caCert, caKey, err := certs.CreateCA("Hub CA", validityDays)
	if err != nil {
		return err
	}
	err = certs.SaveX509CertToPEM(caCert, caCertPath)
	if err == nil {
		err = caKey.ExportPrivateToFile(caKeyPath)
	}

	slog.Info("Generated CA certificate", "caCertPath", caCertPath, "caKeyPath", caKeyPath)
	return err
}
