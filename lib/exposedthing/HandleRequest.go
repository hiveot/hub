package exposedthing

import (
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/lib/messaging"
)

type ThingAction struct {
	Name    string
	Handler func(message *messaging.RequestMessage) *messaging.ResponseMessage
	Input   *td.DataSchema // reuse common schemas
	Output  *td.DataSchema
}
