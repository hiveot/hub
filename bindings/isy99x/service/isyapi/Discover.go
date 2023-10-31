package isyapi

import (
	"fmt"
	"log/slog"
)

// Discover any ISY99x gateway on the local network for 3 seconds.
// If successful this returns the ISY99x gateway address, otherwise an error is returned.
func Discover(timeoutSec int) (addr string, err error) {
	slog.Info("Starting discovery")
	return "", fmt.Errorf("not yet implemented")
}
