package service

import (
	"fmt"
	"log/slog"
	"os"
	"syscall"
	"time"
)

// Stop a process, wait until timeout
// This first tries using SIGTERM. If the timeout expires then try Kill.
// If kill also fails then return an error.
// Returns nil if the process is not running after the timeout
// This returns an error when the process fails to stop after the timeout.
func Stop(name string, pid int) error {
	var err error

	process, _ := os.FindProcess(pid)
	if process == nil {
		return nil
	}
	// be nice about it
	err = process.Signal(syscall.SIGTERM)

	// not clear what to do with err in this case
	_ = err

	// Give the processes time to finish, wait for up to 1 second
	for i := 0; i < 1000; i += 10 {
		// if signal is 0, no signal is sent but error checking is still performed.
		err = process.Signal(syscall.Signal(0))
		if err != nil {
			// error occurs if process no longer running
			break
		}
		// still running
		time.Sleep(time.Millisecond * 10)
	}

	// Check that PID is no longer running
	// On Linux FindProcess always succeeds
	err = nil
	// if signal is 0, no signal is sent but error checking is still performed.
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		// since sigterm fails, the gloves come off
		// This can lead to orphaned child processes though.
		// FIXME: wait with timeout to kill the process
		err = process.Kill()
		msg := fmt.Sprintf(
			"Stopping service '%s' with PID %d failed. Attempt a kill: %s", name, pid, err)
		slog.Error(msg)
	} else {
		// the error confirms that the process has ended
		err = nil
	}

	return err
}
