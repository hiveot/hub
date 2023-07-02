package main

// Launch the hub core
//
// commandline:  hubcore command options
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
//
// hub.yaml also defines the pubsub system to use. Currently only nats is supported.
func main() {
	// parse commandline
	// execute command
}
