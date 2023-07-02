package hubclient

import (
	"errors"
	"github.com/hiveot/hub/api/go/hub"
	"time"
)

func (hc *HubClient) AddUser(userID string, name string, password string, validity time.Duration) (err error) {
	return errors.New("not implemented")
}

// GetProfile returns a client's profile
func (hc *HubClient) GetProfile(clientID string) (profile hub.ClientProfile, err error) {
	return profile, errors.New("not implemented")
}

// ListClients provide a list of known clients and their info
func (hc *HubClient) ListClients() (profiles []hub.ClientProfile, err error) {
	return nil, errors.New("not implemented")

}

// RemoveClient removes a client and disables authentication
// Existing tokens are immediately expired (tbd)
func (hc *HubClient) RemoveClient(clientID string) error {
	return errors.New("not implemented")

}

// ResetPassword reset a user's login password
func (hc *HubClient) ResetPassword(clientID string, password string) (err error) {
	return errors.New("not implemented")
}
