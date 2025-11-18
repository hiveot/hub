package exposedthing

import (
	"github.com/hiveot/gocore/messaging"
	"github.com/hiveot/gocore/wot/td"
)

type ThingAction struct {
	Name    string
	Handler func(message *messaging.RequestMessage) *messaging.ResponseMessage
	Input   *td.DataSchema // reuse common schemas
	Output  *td.DataSchema
}
