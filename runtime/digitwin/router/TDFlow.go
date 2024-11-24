// Package service with digital twin event handling functions
package router

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/wot/transports"
)

// HandleUpdateTD agent updates a TD.
// This converts the operation in an action for the directory service.
func (svc *DigitwinRouter) HandleUpdateTD(msg *transports.ThingMessage) {

	dirMsg := *msg
	dirMsg.ThingID = digitwin.DirectoryDThingID
	dirMsg.Name = digitwin.DirectoryUpdateTDMethod
	_ = svc.digitwinAction(&dirMsg)
}

// HandleReadTD consumer reads a TD
// This converts the operation in an action for the directory service.
func (svc *DigitwinRouter) HandleReadTD(msg *transports.ThingMessage) transports.RequestStatus {
	dirMsg := *msg
	dirMsg.ThingID = digitwin.DirectoryDThingID
	dirMsg.Name = digitwin.DirectoryActionReadTD
	stat := svc.digitwinAction(&dirMsg)
	return stat
}

// HandleReadAllTDs consumer reads all TDs
// This converts the operation in an action for the directory service.
func (svc *DigitwinRouter) HandleReadAllTDs(msg *transports.ThingMessage) transports.RequestStatus {
	dirMsg := *msg
	dirMsg.ThingID = digitwin.DirectoryDThingID
	dirMsg.Name = digitwin.DirectoryActionReadAllTDs
	stat := svc.digitwinAction(&dirMsg)
	return stat
}
