package main

import (
	"flag"
	"fmt"
	"github.com/hiveot/hub/api/go/launcher"
	"github.com/hiveot/hub/core/launcher/config"
	"github.com/hiveot/hub/core/launcher/service"
	"github.com/hiveot/hub/lib/hubcl"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"os"
)

// Connect the launcher service
func main() {
	var cfgFileName = launcher.ServiceName + ".yaml"
	logging.SetLogging("info", "")
	f := utils.GetFolders("", false)
	defaultHomeDir := f.Home

	// handle commandline options
	flag.StringVar(&f.Home, "home", f.Home, "Application home directory")
	flag.StringVar(&cfgFileName, "c", cfgFileName, "Service config filename")
	flag.Usage = func() {
		fmt.Println("Usage: launcher [options] ")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
	}
	flag.Parse()
	// reload f if home changed
	if defaultHomeDir != f.Home {
		f = utils.GetFolders(f.Home, false)
	}

	// load config
	cfg := config.NewLauncherConfig()
	err := f.LoadConfig(cfgFileName, &cfg)
	if err != nil {
		slog.Error("Failed loading launcher config: ", "err", err)
		os.Exit(1)
	}

	// start the launcher but do not connect yet as the message bus core is a plugin.
	svc := service.NewLauncherService(f, cfg)
	err = svc.Start()
	if err != nil {
		slog.Error("Failed starting launcher: ", "err", err)
		os.Exit(1)
	}

	// on successful start of services, connect to the hub to handle rpc requests
	hc, err := hubcl.ConnectToHub("", launcher.ServiceName, f.Certs, "")
	if err != nil {
		slog.Error("Failed connecting to the Hub", "err", err)
		os.Exit(1)
	}
	err = svc.StartListener(hc)

	// wait for a stop signal
	service.WaitForSignal()
	err = svc.Stop()
	if err != nil {
		os.Exit(2)
	}
}
