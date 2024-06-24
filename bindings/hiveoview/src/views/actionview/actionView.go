package actionview

import (
	"github.com/hiveot/hub/lib/hubclient"
	thing "github.com/hiveot/hub/lib/things"
)

// ActionViewData with data for the action window
type ActionViewData struct {
	// The thing the action belongs to
	thingID string
	// the action being viewed in case of an action
	action thing.ActionAffordance
	// the message with the action
	msg thing.ThingMessage
	// the delivery status. Progress is empty if not yet send
	stat hubclient.DeliveryStatus
}
