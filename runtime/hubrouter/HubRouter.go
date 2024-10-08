package hubrouter

import (
	"github.com/hiveot/hub/runtime/api"
	service2 "github.com/hiveot/hub/runtime/authn/service"
	service3 "github.com/hiveot/hub/runtime/authz/service"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"sync"
)

// HubRouter implements the action, event and property flows to and from the
// digital twin.
//
// This uses the transport binding to forward actions and write property requests
// to the agents, and event and property updates to the consumer.
type HubRouter struct {
	dtwStore *service.DigitwinStore
	tb       api.ITransportBinding
	// internal services are directly invoked
	dtwAgent   *service.DigitwinAgent
	dtwService *service.DigitwinService
	authnAgent *service2.AuthnAgent
	authzAgent *service3.AuthzAgent

	// in-memory cache of active actions lookup by messageID
	activeCache map[string]*ActionFlowRecord
	// cache map usage mux
	mux sync.Mutex
}

// SetTransport sets the outgoing transport for adding Forms
func (svc *HubRouter) SetTransport(tb api.ITransportBinding) {
	svc.tb = tb
}

// NewHubRouter instantiates a new hub messaging router
// Use SetTransport to link to a transport for forwarding messages to
// agents and consumers.
//
//	dtwStore is used to update the digital twin status
//	tb is the transport binding for forwarding service requests
func NewHubRouter(
	dtwService *service.DigitwinService,
	dirAgent *service.DigitwinAgent,
	authnAgent *service2.AuthnAgent,
	authzAgent *service3.AuthzAgent,
) *HubRouter {
	ar := &HubRouter{
		dtwStore:    dtwService.DtwStore,
		tb:          nil,
		authnAgent:  authnAgent,
		authzAgent:  authzAgent,
		dtwAgent:    dirAgent,
		dtwService:  dtwService,
		activeCache: make(map[string]*ActionFlowRecord),
	}
	return ar
}
