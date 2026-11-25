package internal

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hub/bindings/isy99x"
	"github.com/hiveot/hub/bindings/isy99x/config"
)

// IsyBinding is the protocol binding for managing the ISY99x Insteon gateway
type IsyBinding struct {
	*thing.ExposedThing

	// Service ID under which it is connected
	// used as binding thingID and as the base for node thingIDs
	bindingID string

	// The ISY99x gateway managed by this binding
	isyGW *IsyGateway
}

// MakeBindingTD generates a TD document for this binding containing properties,
// event and action definitions.
func (svc *IsyBinding) CreateBindingTD() *wot.TD {
	tdoc := wot.NewTD(svc.GetID(), "ISY99x binding", vocab.DeviceTypeService)

	// binding attributes
	prop := tdoc.AddPropertyAsString("loginName", "Login name", "ISY99x gateway login name.")
	prop.ReadOnly = false
	//
	prop = tdoc.AddPropertyAsString("password", "Password", "ISY99x gateway password")
	prop.ReadOnly = false
	prop.WriteOnly = true
	//
	prop = tdoc.AddProperty(vocab.PropNetAddress, "Network Address", "", wot.DataTypeString).
		SetAtType(vocab.PropNetAddress)
	prop.Description = "ISY99x gateway IP address; empty to auto discover."
	prop.ReadOnly = false

	// binding events
	tdoc.AddEvent(vocab.PropNetConnection,
		"Connection changed", "Connection with ISY gateway has changed",
		&wot.DataSchema{Type: wot.DataTypeBool},
	).SetAtType(vocab.PropNetConnection)

	// no binding actions
	return tdoc
}

// Start the ISY99x protocol binding.
// This publishes a TD for this binding, starts a background polling heartbeat.
func (svc *IsyBinding) Start() {
	slog.Info("Starting Isy99x binding")

	tdoc := svc.CreateBindingTD()
	svc.WriteTD(tdoc)

	// the gateway does the real work
	svc.isyGW.Start()
}

// Stop the heartbeat and remove subscriptions
// This does not close the given hubclient connection.
func (svc *IsyBinding) Stop() {
	slog.Info("Stopping the ISY99x Binding")

	svc.isyGW.Stop()
}

// NewIsyBinding creates a new instance of the ISY99x protocol binding service
//
// This creates the Things for the binding, which is responsible for passing
// requests from and notifications to the server.
//
//	bindingID is the binding ThingID. Typically the app clientID.
//	 Use "" for the cell type default if only a single instance is used.
//	cfg configures gateway access
func NewIsyBinding(bindingID string, cfg *config.Isy99xConfig) (*IsyBinding, error) {
	var err error

	if bindingID == "" {
		bindingID = isy99x.Isy99xCellType
	}
	slog.Info("New ISY99x binding")

	// The isy gateway does the heavy lifting
	isyGW, err := NewIsyGateway(bindingID, cfg)

	svc := &IsyBinding{
		ExposedThing: thing.NewExposedThing(bindingID, nil),
		bindingID:    bindingID, // base for thingIDs
		isyGW:        isyGW,
		// prodMap:      prodMap,
		// isyThings:    make(map[string]IIsyThing),
	}

	return svc, err
}
