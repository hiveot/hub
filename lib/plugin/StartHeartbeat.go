package plugin

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// StartHeartbeat runs a heartbeat process in the background.
// This returns a stop function to end the heartbeat loop.
//
// If the heartbeat function is running then the heartbeat will wait until it completes.
// If the heartbeat isn't running it is stopped immediately.
//
// The timer pauses when the heartbeat function is running. If there is a temporary
// delay in the heartbeat there is no risk of overrun.
//
//	interval is the time to wait in between heartbeat invocations
//	fn is the heartbeat function
func StartHeartbeat(interval time.Duration, fn func()) (stopFn func()) {
	// lock to wait until heartbeat has ended
	stopLock := sync.Mutex{}

	ctx, ctxStop := context.WithCancel(context.Background())

	// stoplock for waiting until the hearthbeat has ended
	stopLock.Lock()
	go func() {
		defer stopLock.Unlock()
		slog.Info("Heartbeat started", "interval", interval)
		for {
			fn()

			// Don't use a timer ticker, simply wait for the interval period or
			// for the heartbeat context to cancel.
			// This approach does two things:
			// 1. The heartbeat only runs if the last run is finished
			// 2. Stop instantly between heartbeats and wait until it is finished
			//    if it was running.
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
	return func() {
		// stop the heartbeat loop
		ctxStop()
		// the lock releases when the heartbeat loop has exited
		stopLock.Lock()
		// immediately unlock as this has completed
		stopLock.Unlock()
	}
}
