// Package service with digital twin event handling functions
package router

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/transports"
)

// HandleUpdateTD agent updates a TD.
// This converts the operation in an action for the directory service.
func (svc *DigitwinRouter) HandleUpdateTD(msg *transports.ThingMessage) (output any, err error) {

	dirMsg := *msg
	dirMsg.ThingID = digitwin.DirectoryDThingID
	dirMsg.Name = digitwin.DirectoryUpdateTDMethod
	return svc.digitwinAction(&dirMsg)
}

// HandleReadTD consumer reads a TD
// This converts the operation in an action for the directory service.
func (svc *DigitwinRouter) HandleReadTD(msg *transports.ThingMessage) (output any, err error) {
	dirMsg := *msg
	dirMsg.ThingID = digitwin.DirectoryDThingID
	dirMsg.Name = digitwin.DirectoryActionReadTD
	return svc.digitwinAction(&dirMsg)
}

// HandleReadAllTDs consumer reads all TDs
// This converts the operation in an action for the directory service.
func (svc *DigitwinRouter) HandleReadAllTDs(msg *transports.ThingMessage) (output any, err error) {
	dirMsg := *msg
	dirMsg.ThingID = digitwin.DirectoryDThingID
	dirMsg.Name = digitwin.DirectoryActionReadAllTDs
	return svc.digitwinAction(&dirMsg)
}
