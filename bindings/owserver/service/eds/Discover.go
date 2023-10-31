// Package eds with EDS OWServer API methods
package eds

import (
	"log/slog"
	"net"
	"time"
)

// Discover any EDS OWServer ENet-2 on the local network for 3 seconds
// This uses a UDP Broadcast on port 30303 as stated in the manual
// If found, this sets the service address for further use
// Returns the address or an error if not found
func Discover(timeoutSec int) (addr string, err error) {
	slog.Info("Starting discovery")
	var addr2 *net.UDPAddr
	// listen
	conn, err := net.ListenPacket("udp4", ":30303")
	if err == nil {
		defer conn.Close()

		addr2, err = net.ResolveUDPAddr("udp4", "255.255.255.255:30303")
	}
	if err == nil {
		_, err = conn.WriteTo([]byte("D"), addr2)
	}
	if err != nil {
		return "", err
	}

	buf := make([]byte, 1024)
	// receive 2 messages, first the broadcast, followed by the response, if there is one
	// wait 3 seconds before giving up
	for {
		_ = conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeoutSec)))
		n, remoteAddr, err := conn.ReadFrom(buf)
		if err != nil {
			slog.Info("Discovery ended without results")
			return "", err
		} else if n > 1 {
			switch rxAddr := remoteAddr.(type) {
			case *net.UDPAddr:
				addr = rxAddr.IP.String()
				slog.Info("Found OwServer",
					slog.String("addr", addr),
					slog.String("record", string(buf[:n])))
				return "http://" + addr, nil
			}
		}
	}
}
