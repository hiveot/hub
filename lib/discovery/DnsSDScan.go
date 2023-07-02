// Package hubclient with client for DNS-SD service discovery
package discovery

import (
	"context"
	"golang.org/x/exp/slog"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

// DnsSDScan scans zeroconf publications on local domain
// The zeroconf library does not support browsing of all services, but a workaround is
// to search the service types with "_services._dns-sd._udp" then query each of the service types.
//
//	serviceType to look for in format "_name._tcp", or "" to discover all service types (not all services)
func DnsSDScan(serviceType string, waitSec int) ([]*zeroconf.ServiceEntry, error) {
	sdDomain := "local"
	mu := &sync.Mutex{}

	if serviceType == "" {
		// https://github.com/grandcat/zeroconf/pull/15
		serviceType = "_services._dns-sd._udp"
	}
	records := make([]*zeroconf.ServiceEntry, 0)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		slog.Error("Failed to create DNS-SD resolver", "err", err)
		return nil, err
	}

	// 'records' channel captures the result
	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			rec := entry.ServiceRecord
			slog.Info("Found service",
				"instance", rec.Instance, "type", rec.ServiceName(),
				"domain", rec.Domain, "ip4", entry.AddrIPv4, slog.Int("port", entry.Port))
			mu.Lock()
			records = append(records, entry)
			mu.Unlock()
		}
		slog.Info("No more entries.")
	}(entries)

	duration := time.Second * time.Duration(waitSec)
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	err = resolver.Browse(ctx, serviceType, sdDomain, entries)
	if err != nil {
		slog.Error("Failed to browse", "err", err)
	}
	<-ctx.Done()
	mu.Lock()
	results := records
	mu.Unlock()

	return results, err
}
