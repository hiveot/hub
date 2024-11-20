package httptransport_test

import (
	"fmt"
	"strings"
)

//const userToken = "usertoken"
//const serviceToken = "servicetoken"

type DummyAuthenticator struct{}

func (d *DummyAuthenticator) CreateSessionToken(
	clientID, sessionID string, validitySec int) (token string) {

	if sessionID == "" {
		sessionID = "123"
	}
	return fmt.Sprintf("%s/%s", clientID, sessionID)
}

func (d *DummyAuthenticator) Login(
	clientID string, password string) (token string, err error) {

	if password == clientPassword && clientID == clientLoginID {
		token = d.CreateSessionToken(clientID, "usersession", 0)
		return token, nil
	} else if password == agentPassword && clientID == agentLoginID {
		token = d.CreateSessionToken(clientID, "servicesession", 0)
		return token, nil
	}
	return token, fmt.Errorf("Invalid login")
}

func (d *DummyAuthenticator) Logout(clientID string) {
}

func (d *DummyAuthenticator) ValidatePassword(clientID string, password string) (err error) {
	return nil
}

func (d *DummyAuthenticator) RefreshToken(
	senderID string, clientID string, oldToken string) (newToken string, err error) {

	tokenClientID, sessionID, err := d.ValidateToken(oldToken)
	if clientID == tokenClientID {
		newToken = d.CreateSessionToken(clientID, sessionID, 0)
	} else {
		err = fmt.Errorf("Invalid token, client or sender")
	}
	return newToken, err
}
func (d *DummyAuthenticator) DecodeSessionToken(token string, signedNonce string, nonce string) (clientID string, sessionID string, err error) {
	return d.ValidateToken(token)
}

func (d *DummyAuthenticator) ValidateToken(token string) (clientID string, sessionID string, err error) {

	parts := strings.Split(token, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("badToken")
	}
	clientID = parts[0]
	sessionID = parts[1]

	return clientID, sessionID, err
}
