package owserver

import "github.com/hiveot/hivekit/go/api"

const OWServerCellType = "owserver"

// API of the OWServer gateway binding
type IOWServer interface {
	api.IHiveCell

	// this binding does not expose additional methods

}
