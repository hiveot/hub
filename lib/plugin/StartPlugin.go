package plugin

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/connect"
	"log/slog"
	"os"
)

type PluginConfig struct {
	LogLevel string `yaml:""`
}

// IPlugin interface of protocol bindings and service plugins
type IPlugin interface {
	// Start the plugin with the given environment settings and hub connection
	//	hc is the hub connection for publishing and subscribing
	Start(hc hubclient.IHubClient) error
	Stop()
}

// StartPlugin implements the boilerplate to launch a plugin based on argv
// and its config. This does not return until a signal is received.
//
// AppEnvironment sets the plugin clientID to the application executable name. It can
// be changed by setting env.ClientID before invoking StartPlugin.
// The plugin clientID is used to connect to the hub and lookup a keys and token files
// with the same name in the env.CertsDir directory.
//
//		plugin is the instance of the plugin with Start and Stop methods.
//		clientID is the client's connect ID. certsDir is the location with the service token
//	 file, primary key, and CA certificate.
//		env is the application environment with clientID, certs directory
func StartPlugin(plugin IPlugin, clientID string, certsDir string) {

	// locate the hub, load CA certificate, load service key and token and connect
	hc, err := connect.ConnectToHub("", clientID, certsDir, "", "")
	if err != nil {
		slog.Error("Failed connecting to the Hub", "err", err)
		os.Exit(1)
	}
	// start the service
	err = plugin.Start(hc)
	if err != nil {
		slog.Error("failed starting service", "err", err.Error())
		os.Exit(1)
	}
	WaitForSignal()
	plugin.Stop()

	os.Exit(0)

}
