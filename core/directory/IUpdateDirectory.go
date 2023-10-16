// Package directory with POGS capability definitions of the directory store.
// Unfortunately capnp does generate POGS types so we need to duplicate them
package directory

// UpdateDirectoryCap is the capability ID to modify the directory
const UpdateDirectoryCap = "updateDirectory"

const RemoveTDMethod = "removeTD"

type RemoveTDArgs struct {
	AgentID string `json:"agentID"`
	ThingID string `json:"thingID"`
}

const UpdateTDMethod = "updateTD"

type UpdateTDArgs struct {
	AgentID string `json:"agentID"`
	ThingID string `json:"thingID"`
	TDDoc   []byte `json:"TDDoc"`
}

//--- Interface

// IUpdateDirectory defines the capability of updating the Thing directory
type IUpdateDirectory interface {

	// RemoveTD removes a TD document from the directory
	RemoveTD(agentID, thingID string) (err error)

	// UpdateTD updates the TD document in the directory
	// If the TD doesn't exist it will be added.
	UpdateTD(agentID, thingID string, tdDoc []byte) (err error)
}
