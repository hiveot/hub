package connections

import (
	"golang.org/x/exp/slices"
	"log/slog"
	"sync"
)

// Subscriptions event/property subscriptions of a consumer connection.
//
// This uses "+" as wildcards
type Subscriptions struct {

	// subscriptions of this connection in the form {dThingID}.{name}
	// This doesn't expect too many subscriptions per connection so use a list
	subscriptions []string

	// mutex for access to subscriptions
	mux sync.RWMutex
}

// IsSubscribed returns true  if this client session has subscribed to
// events or properties from the Thing and name
func (s *Subscriptions) IsSubscribed(dThingID string, name string) bool {
	s.mux.RLock()
	defer s.mux.RUnlock()

	if len(s.subscriptions) == 0 {
		return false
	}
	// wildcards
	dThingWC := "+." + name
	nameWC := dThingID + ".+"
	sub := dThingID + "." + name
	for _, s := range s.subscriptions {
		if s == "+.+" {
			// step 1, full wildcard subscriptions
			return true
		} else if s == dThingWC || s == nameWC {
			// step 1, thing or name wildcard subscriptions
			return true
		} else if s == sub {
			// step 1, exact match subscriptions
			return true
		}
	}
	return false
}

// Subscribe adds a subscription for a thing event/property
func (s *Subscriptions) Subscribe(dThingID string, name string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	if dThingID == "" {
		dThingID = "+"
	}
	if name == "" {
		name = "+"
	}
	subKey := dThingID + "." + name
	s.subscriptions = append(s.subscriptions, subKey)
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
	i := slices.Index(s.subscriptions, subKey)
	if i >= 0 {
		s.subscriptions = slices.Delete(s.subscriptions, i, i+1)
	} else {
		slog.Info("UnobserveProperty/unsubscribe. Subscription not found", "subKey", subKey)
	}
}
