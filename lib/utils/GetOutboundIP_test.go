package utils_test

import (
	"github.com/hiveot/hub/lib/utils"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
)

func TestGetOutboundIP(t *testing.T) {
	addr := utils.GetOutboundIP("")
	assert.NotEmpty(t, addr)
	slog.Info("TestGetOutboundIP", "addr", addr)
}

func TestGetOutboundIPBadAddr(t *testing.T) {
	addr := utils.GetOutboundIP("badaddress")
	assert.Empty(t, addr)
}
