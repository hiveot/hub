package certs

import (
	"context"
	"fmt"
	"log/slog"
	"path"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/urfave/cli/v3"
)

// ViewCertsCommand shows info on the Hub self-signed CA
// This does not require any services to run.
//
//	hubcli vca [--certs=CertFolder] view
func ViewCertsCommand(env **api.AppEnvironment) *cli.Command {

	return &cli.Command{
		Name:     "vcerts",
		Category: "certs",
		Usage:    "View CA, server and client certificate info",

		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				return fmt.Errorf("unexpected argument(s) '%s'", cmd.Args().First())
			}
			err := HandleViewCerts(*env)
			return err
		},
	}
}

// HandleViewCerts shows CA, server and client certificate information
func HandleViewCerts(env *api.AppEnvironment) error {
	caCert, _, err := env.GetCACert()
	if err != nil {
		slog.Error("Unable to load the CA certificate", "err", err)
		return err
	}
	caCertPath := path.Join(env.CertsDir, api.DefaultCaCertFile)
	fmt.Println()
	fmt.Println("Self signed CA certificate path: ", caCertPath)
	env.ShowCertDetails(caCert)

	// TODO: fix hidden dependency on the path used by env
	fmt.Println()
	serverCertPath := path.Join(env.CertsDir, api.DefaultServerName+api.DefaultCertFileSuffix)
	serverTLSCert, err := env.GetServerCert("")
	if err != nil {
		fmt.Println("Server certificate not found")
	} else {
		// TODO: show it as a cert chain
		serverCert := serverTLSCert.Leaf
		fmt.Println("Server certificate path: ", serverCertPath)
		env.ShowCertDetails(serverCert)

		// validate against the cert pool
		_, err = utils.VerifyCert(serverCert, env.GetRootCAs())
		if err != nil {
			fmt.Println("  Verification failed!", err.Error())
		} else {
			fmt.Println("  Verification successful")
		}
	}

	fmt.Println()
	clientTLSCert, err := env.GetClientCert("")
	if err != nil {
		fmt.Printf("Client certificate for '%s' not available\n", env.ClientID)
	} else {

		clientCert := clientTLSCert.Leaf
		clientCertPath := path.Join(env.CertsDir, env.ClientID+api.DefaultCertFileSuffix)
		fmt.Println("Client certificate path: ", clientCertPath)
		env.ShowCertDetails(clientCert)

		// validate against the cert pool
		_, err = utils.VerifyCert(clientCert, env.GetRootCAs())
		if err != nil {
			fmt.Println("  Verification failed!", err.Error())
		} else {
			fmt.Println("  Verification successful")
		}
	}
	return nil
}
