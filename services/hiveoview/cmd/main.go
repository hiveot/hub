package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/api"
	rcdevice_recipe "github.com/hiveot/hivekit/go/factory/recipes/rcdevice"
	"github.com/hiveot/hub/services/hiveoview/config"
	"github.com/hiveot/hub/services/hiveoview/src/service"
)

const defaultServerPort = 8443

// const serverCertFile = api.DefaultServerCertFile

// FYI, not all browsers support certificates with ed25519 keys, so this file contains a ecdsa key
// const serverKeyFile = api.DefaultServerKeyFile
const TemplateRootPath = "services/hiveoview/src"

// hiveoview will use the 'hubKey.pem' key and hubCert.pem for the server cert
// and caCert.pem for connecting to the hub message bus.
//
// Note that the service requires a server cert, server CA and a client auth token.
//
// The launcher will automatically generate an auth token in the certs directory
// with the service name.
// To generate a token manually the hubcli can be used:
// "hubcli gentoken __hiveoview", which generates token in: certs/__hiveoview.token
// If the test user __hiveoview doesn't exist it will be added and a private key
// generated.
//
// By default the templates are embedded. For development it can be useful to reload
// the templates for each request. Use the --extfs flag for this.
func main() {
	// var signingKey crypto.PrivateKey
	serverPort := defaultServerPort
	extfs := false

	flag.IntVar(&serverPort, "defaultServerPort", serverPort, "Webserver listening defaultServerPort")
	flag.BoolVar(&extfs, "extfs", extfs, "Use external gohtml filesystem")
	env := api.NewHiveEnvironment("", true)

	// this config will be replaced with hiveoview Thing config
	cfg := config.NewHiveoviewConfig(serverPort)
	_ = env.LoadConfig(env.AppID, &cfg)
	storageDir := env.GetStorageDir(env.AppID)
	err := env.CreateDir(storageDir, 0700)
	if err != nil {
		slog.Error("Unable to create session store", "err", err.Error())
		os.Exit(1)
	}

	// create a hub connection for this service
	// this performs discovery and obtains a directory TDD, and connects to
	// the server using either env.ServerURL or the directory URL.
	r, f, err := rcdevice_recipe.NewRCDeviceRecipe(env, nil)
	_ = f
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}

	// TLS Cert to serve the hiveoview web pages
	// Its private key is used to sign the cookies.
	serverCert, err := env.GetServerCert()
	if err != nil {
		slog.Error("Unable to load server certificate: " + err.Error())
		return
	}

	// development only, serve files and parse templates from filesystem
	rootPath := ""
	if extfs {
		cwd, _ := os.Getwd()
		rootPath = path.Join(cwd, TemplateRootPath)
	}
	// FIXME: currently timeout is set to 20 sec to allow slow requests to complete.
	// However these requests should handle multiple async updates instead.
	// for example zwavejs refresh device info can take up to 20sec
	if err != nil {
		slog.Error("Unable to load server certificate: " + err.Error())
		return
	}

	// The client's URL is used to connect by all sessions.
	hubURL := r.GetClient().GetConnectURL()
	env.ServerURL = hubURL
	svc, err := service.NewHiveovService(env,
		cfg.ServerPort, false, rootPath, serverCert, env.GetRootCAs(),
		time.Second*20, storageDir)
	r.SetRequestSink(svc)
	svc.SetNotificationSink(r)

	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}
	r.Run()
}
