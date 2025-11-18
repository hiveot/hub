package service_test

import (
	"log/slog"
	"testing"

	"github.com/hiveot/hivehub/bindings/ipnet/service"
	"github.com/hiveot/hivekitgo/utils/net"
	"github.com/stretchr/testify/assert"
)

func TestNmapScan(t *testing.T) {
	nmap := service.NewNmapAPI()
	subnets, err := net.GetIP4Subnets(true)
	assert.NoError(t, err)
	devices, err := nmap.ScanSubnet(subnets, false, false) // no port scan and no root
	assert.NoError(t, err)
	slog.Info("Nodes found nodes in the nmap device table", "count", len(devices))
	assert.NotEmpty(t, devices, "Expected nodes on the network")
}
