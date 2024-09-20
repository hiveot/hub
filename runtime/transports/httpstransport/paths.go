package httpstransport

// Paths used by this protocol binding
//
// These are used in generating the protocol binding Forms in the TDD and in the
// http router.
const (
	// Form paths for use by consumers; thingID is the digital twin thing ID
	ConnectSSEPath                 = "/sse"
	GetReadAllPropertiesPath       = "/digitwin/properties/{thingID}/+"
	GetReadPropertyPath            = "/digitwin/properties/{thingID}/{name}"
	PostWritePropertyPath          = "/digitwin/properties/{thingID}/{name}"
	PostObserveAllPropertiesPath   = "/digitwin/observe/{thingID}/+"
	PostObservePropertyPath        = "/digitwin/observe/{thingID}/{name}"
	PostUnobserveAllPropertiesPath = "/digitwin/unobserve/{thingID}/+"
	PostUnobservePropertyPath      = "/digitwin/unobserve/{thingID}/{name}"

	GetReadAllEventsPath         = "/digitwin/events/{thingID}/+"
	PostSubscribeAllEventsPath   = "/digitwin/subscribe/{thingID}/+"
	PostSubscribeEventPath       = "/digitwin/subscribe/{thingID}/{name}"
	PostUnsubscribeAllEventsPath = "/digitwin/unsubscribe/{thingID}/+"
	PostUnsubscribeEventPath     = "/digitwin/unsubscribe/{thingID}/{name}"

	GetReadAllActionsPath = "/digitwin/actions/{thingID}/+"
	GetReadActionPath     = "/digitwin/actions/{thingID}/{name}"
	PostInvokeActionPath  = "/digitwin/actions/{thingID}/{name}"

	GetReadAllThingsPath = "/digitwin/directory/+" // query param offset=, limit=
	GetReadThingPath     = "/digitwin/directory/{thingID}"

	// authentication service
	PostLoginPath   = "/authn/login"
	PostLogoutPath  = "/authn/logout"
	PostRefreshPath = "/authn/refresh"

	// Form paths for use by agents/servients
	PostAgentUpdatePropertyPath = "/agent/property/{thingID}/{name}"
	PostAgentUpdateTDDPath      = "/agent/tdd/{thingID}"
	PostAgentPublishEventPath   = "/agent/event/{thingID}/{name}"
)
