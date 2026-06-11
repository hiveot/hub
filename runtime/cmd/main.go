package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/runtime"
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
	env := factory.NewAppEnvironment("", true)
	utils.SetLogging(env.LogLevel, "")
	fmt.Println("home: ", env.HomeDir)
	if len(flag.Args()) > 0 {
		println("ERROR: No arguments expected.")
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Initialize the runtime configuration, directories and load keys and certificates
	r := runtime.NewRuntime(env)
	err := r.Start()
	if err != nil {
		println("Starting hivot runtime failed: ", err.Error())
		os.Exit(1)
	}

	// Startup Successful
	// wait until signal
	utils.WaitForSignal()

	println("Graceful shutdown of Runtime")

	r.Stop()

	// give background tasks time to stop
	time.Sleep(time.Millisecond * 100)
}
