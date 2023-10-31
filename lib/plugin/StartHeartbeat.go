package plugin

import (
	"context"
	"log/slog"
	"time"
)

// StartHeartbeat runs a heartbeat process in the background.
// This returns a stop function to end the heartbeat loop.
// If the heartbeat function is running then the heartbeat will wait until it completes.
//
// The timer pauses when the heartbeat function is running. If there is a temporary
// delay in the heartbeat there is no risk of overrun.
//
//	interval is the time to wait in between heartbeat invocations
//	fn is the heartbeat function
func StartHeartbeat(interval time.Duration, fn func()) (stopFn func()) {

	ctx, stopFn := context.WithCancel(context.Background())
	go func() {
		slog.Info("Heartbeat started", "interval", interval)
		for {
			fn()

			// Don't use a timer ticker. If the heartbeat function is slow we don't
			// want it to overrun and cause multiple instances to run.
			timer, cancelFn := context.WithTimeout(context.Background(), interval)
			select {
			case <-timer.Done():
				cancelFn()
			case <-ctx.Done():
				cancelFn()
				slog.Info("Heartbeat stopped")
				return
			}
		}
	}()
	return stopFn
}
