package net_test

import (
	"github.com/hiveot/hub/lib/net"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
)

func TestGetOutboundIP(t *testing.T) {
	addr := net.GetOutboundIP("")
	assert.NotEmpty(t, addr)
	slog.Info("TestGetOutboundIP", "addr", addr)
}

func TestGetOutboundIPBadAddr(t *testing.T) {
	addr := net.GetOutboundIP("badaddress")
	assert.Empty(t, addr)
}
