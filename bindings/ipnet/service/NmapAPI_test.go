package service_test

import (
	"github.com/hiveot/hub/bindings/ipnet/service"
	"github.com/hiveot/hub/transports/tputils/net"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
)

func TestNmapScan(t *testing.T) {
	nmap := service.NewNmapAPI()
	subnets, err := net.GetIP4Subnets()
	assert.NoError(t, err)
	devices, err := nmap.ScanSubnet(subnets, false, false) // no port scan and no root
	assert.NoError(t, err)
	slog.Info("Nodes found nodes in the nmap device table", "count", len(devices))
	assert.NotEmpty(t, devices, "Expected nodes on the network")
}
