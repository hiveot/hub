package exposedthing

import (
	"github.com/hiveot/hivekitgo/messaging"
	"github.com/hiveot/hivekitgo/wot/td"
)

type ThingAction struct {
	Name    string
	Handler func(message *messaging.RequestMessage) *messaging.ResponseMessage
	Input   *td.DataSchema // reuse common schemas
	Output  *td.DataSchema
}
