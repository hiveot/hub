package certs

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/cells/certs"
	"github.com/hiveot/hivekit/go/cells/certs/certutils"
	"github.com/urfave/cli/v3"
)

// ViewCACommand shows info on the Hub self-signed CA
// This does not require any services to run.
//
//	hubcli vca [--certs=CertFolder] view
func ViewCACommand(certsFolder *string) *cli.Command {

	return &cli.Command{
		Name:     "vca",
		Category: "certs",
		Usage:    "View CA and server certificate info",

		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				return fmt.Errorf("unexpected argument(s) '%s'", cmd.Args().First())
			}
			err := HandleViewCACert(*certsFolder)
			return err
		},
	}
}

// HandleViewCACert shows CA certificate information
func HandleViewCACert(certsFolder string) error {
	caCertPath := path.Join(certsFolder, certs.DefaultCaCertFile)

	caCert, err := certutils.LoadX509CertFromPEM(caCertPath)
	if err != nil {
		slog.Error("Unable to load the CA certificate", "err", err)
		return err
	}
	fmt.Println("CA certificate path: ", caCertPath)
	fmt.Println("  IsCA       : ", caCert.IsCA)
	fmt.Println("  Version    : ", caCert.Version)
	fmt.Println("  Valid until: ", caCert.NotAfter.Format(time.RFC1123Z))
	fmt.Println("  Subject    : ", caCert.Subject.String())
	fmt.Println("  Issuer     : ", caCert.Issuer.String())
	fmt.Println("  DNS names  : ", caCert.DNSNames)
	fmt.Println()

	hubCertPath := path.Join(certsFolder, "hubCert.pem")
	serverCert, err := certutils.LoadX509CertFromPEM(hubCertPath)
	if err != nil {
		slog.Error("Unable to load the server certificate", "err", err)
		return err
	}
	fmt.Println("Server certificate path: ", hubCertPath)
	fmt.Println("  IsCA        : ", serverCert.IsCA)
	fmt.Println("  Version     : ", serverCert.Version)
	fmt.Println("  Valid until : ", serverCert.NotAfter.Format(time.RFC1123Z))
	fmt.Println("  Subject     : ", serverCert.Subject.String())
	fmt.Println("  Issuer      : ", serverCert.Issuer.String())
	fmt.Println("  DNS names   : ", serverCert.DNSNames)
	fmt.Println("  IP addresses: ", serverCert.IPAddresses)
	return nil
}
