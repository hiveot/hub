// Package internal handles node configuration commands
package internal

import (
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// HandleConfigCommand for handling node configuration changes
// Not supported
func (app *IsyApp) HandleConfigCommand(nodeHWID string, config types.NodeAttrMap) {
	logrus.Infof("IsyApp.HandleConfigCommand for node HWID '%s'", nodeHWID)
	// at this moment no ISY configuration is changed so just pass it on
	app.pub.UpdateNodeConfigValues(nodeHWID, config)
}
