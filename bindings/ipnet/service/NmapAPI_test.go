package service_test

import (
	"log/slog"
	"testing"

	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/ipnet/service"
	"github.com/stretchr/testify/assert"
)

func TestNmapScan(t *testing.T) {
	nmap := service.NewNmapAPI()
	subnets, err := utils.GetIP4Subnets(true)
	assert.NoError(t, err)
	devices, err := nmap.ScanSubnet(subnets, false, false) // no port scan and no root
	assert.NoError(t, err)
	slog.Info("Nodes found nodes in the nmap device table", "count", len(devices))
	assert.NotEmpty(t, devices, "Expected nodes on the network")
}
