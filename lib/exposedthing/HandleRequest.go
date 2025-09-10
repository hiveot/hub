package exposedthing

import (
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/wot/td"
)

type ThingAction struct {
	Name    string
	Handler func(message *messaging.RequestMessage) *messaging.ResponseMessage
	Input   *td.DataSchema // reuse common schemas
	Output  *td.DataSchema
}
