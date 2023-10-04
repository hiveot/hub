package main

import (
	"github.com/hiveot/hub/core/launcher/config"
	"github.com/hiveot/hub/core/launcher/service"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"os"
)

// Connect the launcher service
func main() {
	//launcherID := path.Base(os.Args[0])
	//var cfgFileName = launcherID + ".yaml"

	env := utils.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")

	//defaultHomeDir := f.HomeDir
	//
	//// handle commandline options
	//flag.StringVar(&f.HomeDir, "home", f.HomeDir, "Application home directory")
	//flag.StringVar(&cfgFileName, "c", cfgFileName, "Service config filename")
	//flag.Usage = func() {
	//	fmt.Println("Usage: launcher [options] ")
	//	fmt.Println()
	//	fmt.Println("Options:")
	//	flag.PrintDefaults()
	//}
	//flag.Parse()
	//// reload f if home changed
	//if defaultHomeDir != f.HomeDir {
	//	f = utils.GetAppEnvironment(f.HomeDir)
	//}
	//
	// load config
	cfg := config.NewLauncherConfig()
	err := env.LoadConfig(env.ConfigFile, &cfg)
	if err != nil {
		slog.Error("Failed loading launcher config: ", "err", err)
		os.Exit(1)
	}

	// start the launcher but do not connect yet as the core can be started by the launcher itself.
	// the core will generate the launcher key and token.
	svc := service.NewLauncherService(env, cfg, nil)
	err = svc.Start()
	if err != nil {
		slog.Error("Failed starting launcher: ", "err", err)
		os.Exit(1)
	}

	// wait for a stop signal
	service.WaitForSignal()
	err = svc.Stop()
	if err != nil {
		os.Exit(2)
	}
}
