package utils

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ExitOnSignal starts a background process and closes the context when a SIGINT or SIGTERM is received.
//
// If a release function is provided it is invoked before the function exits normally.
// If no release function is provided this ends the process with os.Exit(0).
// This returns a child context which is cancelled before the call to release.
func ExitOnSignal(release func()) context.Context {

	exitCtx, cancelFn := context.WithCancel(context.Background())
	go func() {
		WaitForSignal(exitCtx)

		// cancel the context. This should invoke Done()
		cancelFn()

		// if a release function is provided, it handles its own exit, otherwise exit here
		if release != nil {
			release()
		} else {
			time.Sleep(time.Millisecond)
			os.Exit(0)
		}
	}()
	return exitCtx
}

// WaitForSignal waits until a SIGINT or SIGTERM is received or the context is closed
func WaitForSignal(ctx context.Context) {

	// catch all signals since not explicitly listing
	exitChannel := make(chan os.Signal, 1)

	signal.Notify(exitChannel, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-exitChannel:
	case <-ctx.Done():
	}
	return
}
