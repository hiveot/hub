package main

import (
	"fmt"
	"os"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/runtime/pkg"
)

// Launch the hub digital twin runtime.
// This starts the digital twin stores and the protocol servers.
//
// commandline:  runtime [options]
//
// This runs
func main() {

	env := api.NewHiveEnvironment("", true)
	utils.SetLogging(env.LogLevel, "")
	hub, f, err := pkg.NewHubRuntime(env)

	if err != nil {
		fmt.Println("Runtime startup failed: " + err.Error())
		os.Exit(1)
	}
	fmt.Printf("main: homeDir: %s\n", env.HomeDir)
	fmt.Printf("main: Hub is running and listening on '%v'\n", f.GetConnectURLs())

	// Startup Successful
	// wait until signal
	utils.WaitForSignal()

	println("Graceful shutdown of Runtime")

	hub.Stop()

}
