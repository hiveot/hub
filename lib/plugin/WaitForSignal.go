package plugin

import (
	"os"
	"os/signal"
	"syscall"
)

// WaitForSignal waits until a SIGINT or SIGTERM is received
func WaitForSignal() {

	// catch all signals since not explicitly listing
	exitChannel := make(chan os.Signal, 1)

	signal.Notify(exitChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	sigID := <-exitChannel
	println("Exiting with signal (", sigID, "): ", os.Args[0], "\n")
}
