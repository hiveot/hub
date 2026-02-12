package connections

import (
	"sync"
)

// Subscriptions event/property subscriptions of a consumer connection.
//
// This uses "+" as wildcards
type Subscriptions struct {

	// map of subscriptions to correlationID
	// subscriptions of this connection in the form {dThingID}.{name}
	// not many are expected.
	subscriptions map[string]string

	// mutex for access to subscriptions
	mux sync.RWMutex
}

// GetSubscription returns the correlation ID if this client session has subscribed to
// events or properties from the Thing and name.
// If the subscription is unknown then return an empty string.
func (s *Subscriptions) GetSubscription(thingID string, name string) string {
	s.mux.RLock()
	defer s.mux.RUnlock()

	if len(s.subscriptions) == 0 {
		return ""
	}
	// wildcards
	thingWC := "+." + name
	nameWC := thingID + ".+"
	sub := thingID + "." + name
	for k, v := range s.subscriptions {
		if k == "+.+" {
			// step 1, full wildcard subscriptions
			return v
		} else if k == thingWC || k == nameWC {
			// step 1, thing or name wildcard subscriptions
			return v
		} else if k == sub {
			// step 1, exact match subscriptions
			return v
		}
	}
	return ""
}

// IsSubscribed returns true  if this client session has subscribed to
// events or properties from the Thing and name
func (s *Subscriptions) IsSubscribed(thingID string, name string) bool {
	corrID := s.GetSubscription(thingID, name)
	return corrID != ""
}

// Subscribe adds a subscription for a thing event/property
func (s *Subscriptions) Subscribe(thingID string, name string, correlationID string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	if thingID == "" {
		thingID = "+"
	}
	if name == "" {
		name = "+"
	}
	subKey := thingID + "." + name
	if s.subscriptions == nil {
		s.subscriptions = make(map[string]string)
	}
	s.subscriptions[subKey] = correlationID
}

// Unsubscribe removes a subscription for a thing event/property
func (s *Subscriptions) Unsubscribe(dThingID string, name string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	if dThingID == "" {
		dThingID = "+"
	}
	if name == "" {
		name = "+"
	}
	subKey := dThingID + "." + name
	delete(s.subscriptions, subKey)
}
