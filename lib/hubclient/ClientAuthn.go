package hubclient

import "errors"

func (hc *HubClient) Login(clientID string, password string) (authToken string, err error) {
	return "", errors.New("not implemented")
}

// Logout invalidates the authentication token and requires a login
func (hc *HubClient) Logout(userID string, refreshToken string) (err error) {
	return errors.New("not implemented")
}

// Refresh a short lived authentication token.
//
//	clientID is the userID, deviceID or serviceID whose token to refresh.
//	oldToken must be a valid token obtained at login or refresh
//
// This returns a short lived auth token that can be used to authenticate with the hub
// This fails if the token has expired or does not belong to the clientID
func (hc *HubClient) Refresh(clientID string, oldToken string) (newToken string, err error) {
	return "", errors.New("not implemented")

}

// UpdateName updates a user's name
func (hc *HubClient) UpdateName(clientID string, name string) (err error) {
	return errors.New("not implemented")
}

// UpdatePassword changes the client password
// Login or Refresh must be called successfully first.
func (hc *HubClient) UpdatePassword(clientID string, newPassword string) error {
	return errors.New("not implemented")
}
