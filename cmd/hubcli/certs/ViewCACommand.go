package certs

import (
	"fmt"
	"log/slog"
	"path"
	"time"

	"github.com/hiveot/gocore/certs"
	"github.com/urfave/cli/v2"
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

		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() > 0 {
				return fmt.Errorf("unexpected argument(s) '%s'", cCtx.Args().First())
			}
			err := HandleViewCACert(*certsFolder)
			return err
		},
	}
}

// HandleViewCACert shows CA certificate information
func HandleViewCACert(certsFolder string) error {
	caCertPath := path.Join(certsFolder, certs.DefaultCaCertFile)

	caCert, err := certs.LoadX509CertFromPEM(caCertPath)
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
	serverCert, err := certs.LoadX509CertFromPEM(hubCertPath)
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
