package certs

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/utils"
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
	caCertPath := path.Join(certsFolder, api.DefaultCaCertFile)

	caCert, err := utils.LoadCACert(caCertPath)
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
	x509Cert, err := utils.LoadX509Cert(hubCertPath)
	if err != nil {
		slog.Error("Unable to load the server certificate", "err", err)
		return err
	}
	// TODO: show it as a cert chain
	serverCert := x509Cert[0]
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
