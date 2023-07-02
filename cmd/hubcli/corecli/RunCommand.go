package corecli

import (
	"fmt"
)
import "github.com/urfave/cli/v2"

// Launch the hub core
//
// commandline:  hubcore start
//
// commands:
//
//	init     initialize the core from scratch
//	run      run the hub core. core must have been initialized before it can run
//
// options:
//
//	--config=file   use the given config file. This defines the folder structure.
//	--force         use with init, force overwriting existing configuration and data
//	--home=folder   use the given home folder instead of the base of the application binary
//
// init:
//  1. creates missing folders (see below)
//     if --config=hub.yaml is provided then use this file for the folders
//  2. generate new core system config file(s) in the config folder
//     keep existing files, unless --force is used
//  2. create new self-signed CA and server certificates
//     keep existing files unless --force is used
//  4. initialize the core system storage in the stores folder
//     keep existing storage, unless --force is used
//  5. creates an admin account for the local user
//     keep existing admin if any, unless --force is used
//
// init creates a config file $home/config/hub.yaml with the following folder structure,
// where $home is the directory of the application installation folder:
//
//	$home/bin application binary
//	$home/plugins contains additional application plugins
//	$home/config  configuration files for core and plugins
//	$home/stores  storage of directory and history database
//	$home/certs with server and CA certificates

// RunCommand runs the Hub core services
//
//	hubcli run [--certs=CertFolder]  [--hostname=hostname]
func RunCommand(certsFolder string) *cli.Command {

	return &cli.Command{
		Name:     "run",
		Usage:    "Run the hub core services",
		Category: "core",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() > 0 {
				return fmt.Errorf("unexpected argument(s) '%s'", cCtx.Args().First())
			}
			err := HandleRunCore(certsFolder)
			return err
		},
	}
}

// HandleRunCore handles running of the Hub core services
// A CA certificate must exist.
func HandleRunCore(certsFolder string) error {
	//caCertPath := path.Join(certsFolder, certs.DefaultCaCertFile)
	//caKeyPath := path.Join(certsFolder, certs.DefaultCaKeyFile)
	return nil
}
