package utils

import (
	"log/slog"
	"net"
)

// GetOutboundIP returns the default outbound IP address to reach the given hostname.
// This uses 1.1.1.1 as the default destination.
// TODO: use the default gateway address instead so this works without internet access.
//
// Use a local hostname if a subnet other than the default one should be used.
// Use "" for the default route address
//
//	destination to reach or "" to use 1.1.1.1 (no connection will be established)
func GetOutboundIP(destination string) net.IP {
	if destination == "" {
		destination = "1.1.1.1"
	}
	// This dial command doesn't actually create a connection, just determines
	// the connection needed to connect to it.
	conn, err := net.Dial("udp", destination+":80")
	if err != nil {
		slog.Error("dial udp error", "err", err)
		return nil
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}
