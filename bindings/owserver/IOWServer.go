package owserver

import "github.com/hiveot/hivekit/go/api"

const DefaultOWServerThingID = "owserver"

type IOWServer interface {
	api.IHiveCell
}
