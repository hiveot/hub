// Package service with digital twin event handling functions
package router

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/transports"
)

// HandleUpdateTD agent updates a TD.
// This converts the operation in an action for the directory service.
func (svc *DigitwinRouter) HandleUpdateTD(msg *transports.ThingMessage) (
	completed bool, output any, err error) {

	dirMsg := *msg
	dirMsg.ThingID = digitwin.DirectoryDThingID
	dirMsg.Name = digitwin.DirectoryUpdateTDMethod
	output, err = svc.digitwinAction(&dirMsg)
	return true, output, err
}

// HandleReadTD consumer reads a TD
// This converts the operation in an action for the directory service.
func (svc *DigitwinRouter) HandleReadTD(msg *transports.ThingMessage) (
	completed bool, output any, err error) {

	dirMsg := *msg
	dirMsg.ThingID = digitwin.DirectoryDThingID
	dirMsg.Name = digitwin.DirectoryActionReadTD
	output, err = svc.digitwinAction(&dirMsg)
	return true, output, err
}

// HandleReadAllTDs consumer reads all TDs
// This converts the operation in an action for the directory service.
func (svc *DigitwinRouter) HandleReadAllTDs(msg *transports.ThingMessage) (
	completed bool, output any, err error) {
	dirMsg := *msg
	dirMsg.ThingID = digitwin.DirectoryDThingID
	dirMsg.Name = digitwin.DirectoryActionReadAllTDs
	output, err = svc.digitwinAction(&dirMsg)
	return true, output, err
}
