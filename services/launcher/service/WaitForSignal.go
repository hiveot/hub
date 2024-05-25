package service

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// WaitForSignal waits until a TERM or INT signal is received
// Intended for use by hub plugins to run until the app is done
func WaitForSignal() {

	// catch all signals since not explicitly listing
	exitChannel := make(chan os.Signal, 1)

	signal.Notify(exitChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSEGV)

	sig := <-exitChannel
	slog.Warn("RECEIVED SIGNAL", "signal", sig, "for", os.Args[0])
	fmt.Println()
}
