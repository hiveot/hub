package tputils

import (
	"errors"
	"fmt"
	"github.com/teris-io/shortid"
	"strings"
	"time"
)

// DummyAuthenticator for testing the transport protocol bindings
type DummyAuthenticator struct {
	passwords map[string]string
	tokens    map[string]string
}

// AddClient adds a test client and return an auth token
func (d *DummyAuthenticator) AddClient(clientID string, password string) string {
	d.passwords[clientID] = password
	sessionID := clientID + shortid.MustGenerate()
	token := d.CreateSessionToken(clientID, sessionID, 0)
	d.tokens[clientID] = token
	return token
}

//func (d *DummyAuthenticator) AddToken(clientID string, token string) {
//	d.tokens[clientID] = token
//}

func (d *DummyAuthenticator) CreateSessionToken(
	clientID, sessionID string, validity time.Duration) (token string) {

	if sessionID == "" {
		sessionID = shortid.MustGenerate()
	}
	token = fmt.Sprintf("%s/%s", clientID, sessionID)
	// simulate a session with the tokens map
	d.tokens[clientID] = token
	return token
}

func (d *DummyAuthenticator) Login(
	clientID string, password string) (token string, err error) {

	currPass, isClient := d.passwords[clientID]
	if isClient && currPass == password {
		sessionID := clientID + shortid.MustGenerate()
		token = d.CreateSessionToken(clientID, sessionID, 0)
		d.tokens[clientID] = token
		return token, nil
	}
	return "", fmt.Errorf("Invalid login")
}

func (d *DummyAuthenticator) Logout(clientID string) {
	delete(d.passwords, clientID)
	delete(d.tokens, clientID)
}

func (d *DummyAuthenticator) ValidatePassword(clientID string, password string) (err error) {
	currPass, isClient := d.passwords[clientID]
	if isClient && currPass == password {
		return nil
	}
	return errors.New("bad login or pass")
}

func (d *DummyAuthenticator) RefreshToken(
	senderID string, oldToken string) (newToken string, err error) {

	tokenClientID, sessionID, err := d.ValidateToken(oldToken)
	if err != nil || senderID != tokenClientID {
		err = fmt.Errorf("Invalid token, client or sender")
	} else {
		newToken = d.CreateSessionToken(senderID, sessionID, 0)
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
	// simulate a session by checking if a recent token was issued
	_, found := d.tokens[clientID]
	if !found {
		err = errors.New("no active session")
	}

	return clientID, sessionID, err
}

func NewDummyAuthenticator() *DummyAuthenticator {
	d := &DummyAuthenticator{
		passwords: make(map[string]string),
		tokens:    make(map[string]string),
	}
	return d
}
