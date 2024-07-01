package main

import (
	"flag"
	"fmt"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/runtime"
	"os"
	"time"
)

// Launch the hub digital twin runtime.
// This starts the digital twin stores and the protocol servers.
//
// commandline:  runtime [options]
//
// This runs
func main() {
	flag.Usage = func() {
		fmt.Println("Usage: runtime [options]")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
	}
	env := plugin.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")
	fmt.Println("home: ", env.HomeDir)
	if len(flag.Args()) > 0 {
		println("ERROR: No arguments expected.")
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Initialize the runtime configuration, directories and load keys and certificates
	cfg := runtime.NewRuntimeConfig()
	r := runtime.NewRuntime(cfg)
	err := r.Start(&env)
	if err != nil {
		os.Exit(1)
	}

	// Startup Successful
	// wait until signal
	plugin.WaitForSignal()

	println("Graceful shutdown of Runtime")

	r.Stop()

	// give background tasks time to stop
	time.Sleep(time.Millisecond * 100)
}
