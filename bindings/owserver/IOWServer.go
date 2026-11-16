package owserver

import "github.com/hiveot/hivekit/go/api"

const OWServerCellType = "owserver"

type IOWServer interface {
	api.IHiveCell
}
