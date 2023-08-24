package authnservice

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/msgserver"
)

// AuthnUser handles authentication user requests
// This implements the IAuthnUser interface.
//
// This implements the IAuthnUser interface.
type AuthnUser struct {
	// Client record persistence
	store authn.IAuthnStore
	// message server for updating authn
	msgServer msgserver.IMsgServer
	// CA certificate for validating cert
	caCert *x509.Certificate
}

// CreateToken creates an authentication token using the external tokenizer or
// the built-in tokenizer.
// This invokes the external tokenizer if provided and falls-back to the built-in
// tokenizer.
func (svc *AuthnUser) CreateToken(clientID string, clientType string, pubKey string, validitySec int) (newToken string, err error) {
	return svc.msgServer.CreateToken(clientID, clientType, pubKey, validitySec)
}

// GeneratePassword with upper, lower, numbers and special characters
//func (svc *AuthnUser) GeneratePassword(length int, useSpecial bool) (password string) {
//	const charsLow = "abcdefghijklmnopqrstuvwxyz"
//	const charsUpper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
//	const charsSpecial = "!#$%&*+-./:=?@^_"
//	const numbers = "0123456789"
//	var pool = []rune(charsLow + numbers + charsUpper)
//
//	if length < 2 {
//		length = 8
//	}
//	if useSpecial {
//		pool = append(pool, []rune(charsSpecial)...)
//	}
//	rand.Seed(uint64(time.Now().Unix()))
//	//pwchars := make([]string, length)
//	pwchars := strings.Builder{}
//
//	for i := 0; i < length; i++ {
//		pos := rand.Intn(len(pool))
//		pwchars.WriteRune(pool[pos])
//	}
//	password = pwchars.String()
//	return password
//}

// GetProfile returns a client's profile
func (svc *AuthnUser) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	clientProfile, err := svc.store.GetProfile(clientID)
	return clientProfile, err
}

// NewToken validates a password and issues an authn token
func (svc *AuthnUser) NewToken(clientID string, password string) (newToken string, err error) {
	clientProfile, err := svc.store.VerifyPassword(clientID, password)
	if err != nil {
		return "", err
	}
	if clientProfile.PubKey == "" {
		return "", fmt.Errorf("no public key on file for '%s'", clientID)
	}
	newToken, err = svc.CreateToken(clientID, clientProfile.ClientType, clientProfile.PubKey, clientProfile.ValiditySec)
	return newToken, err
}

// notification handler invoked when clients have been updated
// this invokes a reload of server authn
func (svc *AuthnUser) onChange() {
	_ = svc.msgServer.ApplyAuthn(svc.store.GetEntries())
}

// Refresh issues a new token if the given token is valid
// This returns a refreshed token that can be used to connect to the messaging server
// the old token must be a valid jwt token belonging to the clientID
func (svc *AuthnUser) Refresh(clientID string, oldToken string) (newToken string, err error) {
	// verify the token
	clientProfile, err := svc.store.GetProfile(clientID)
	if err != nil {
		return "", err
	}
	err = svc.msgServer.ValidateToken(clientID, clientProfile.PubKey, oldToken, "", "")
	if err != nil {
		return "", fmt.Errorf("error validating oldToken of client %s: %w", clientID, err)
	}
	newToken, err = svc.CreateToken(clientID, clientProfile.ClientType, clientProfile.PubKey, clientProfile.ValiditySec)
	return newToken, err
}

// UpdateName
func (svc *AuthnUser) UpdateName(clientID string, displayName string) (err error) {
	clientProfile, err := svc.store.GetProfile(clientID)
	clientProfile.DisplayName = displayName
	err = svc.store.Update(clientID, clientProfile)
	// this doesn't affect authentication
	return err
}

func (svc *AuthnUser) UpdatePassword(clientID string, newPassword string) (err error) {
	_, err = svc.GetProfile(clientID)
	if err != nil {
		return err
	}
	err = svc.store.SetPassword(clientID, newPassword)
	if err != nil {
		return err
	}
	svc.onChange()
	return err
}

func (svc *AuthnUser) UpdatePubKey(clientID string, newPubKey string) (err error) {
	clientProfile, err := svc.store.GetProfile(clientID)
	if err != nil {
		return err
	}
	clientProfile.PubKey = newPubKey
	err = svc.store.Update(clientID, clientProfile)
	if err != nil {
		return err
	}
	// run in the background so a response can be sent
	go svc.onChange()
	return err
}

// ValidateToken verifies if the token is valid and belongs to the claimed user
//func (svc *AuthnUser) ValidateToken(clientID string, oldToken string) (err error) {
//	// verify the token
//	entry, err := svc.store.Get(clientID)
//	if err != nil {
//		return err
//	}
//	_ = entry
//	err = svc.tokenizer.ValidateToken(clientID, oldToken, "", "")
//	return err
//}

// ValidateCert verifies that the given certificate belongs to the client
// and is signed by our CA.
// - CN is clientID (todo: other means?)
// - Cert validates against the svc CA
// This is intended for a local setup that use a self-signed CA.
// The use of JWT keys is recommended over certs as this isn't a domain name validation problem.
//func (svc *AuthnUser) ValidateCert(clientID string, clientCertPEM string) error {
//
//	if svc.caCert == nil {
//		return fmt.Errorf("no CA on file")
//	}
//	certBlock, _ := pem.Decode([]byte(clientCertPEM))
//	if certBlock == nil {
//		return fmt.Errorf("invalid cert pem for client '%s. decode failed", clientID)
//	}
//	clientCert, err := x509.ParseCertificate(certBlock.Bytes)
//	if err != nil {
//		return err
//	}
//	// verify the cert against the CA
//	caCertPool := x509.NewCertPool()
//	caCertPool.AddCert(svc.caCert)
//	verifyOpts := x509.VerifyOptions{
//		Roots:     caCertPool,
//		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
//	}
//
//	_, err = clientCert.Verify(verifyOpts)
//
//	// verify the certs belongs to the clientID
//	certUser := clientCert.Subject.CommonName
//	if certUser != clientID {
//		return fmt.Errorf("cert user '%s' doesnt match client '%s'", certUser, clientID)
//	}
//	return nil
//}

// NewAuthnUserService returns a user authentication capability.
//
//	store holds the authentication client records
//	caCert is an optional CA used to verify certificates. Use nil to not authn using client certs
func NewAuthnUserService(
	store authn.IAuthnStore,
	msgServer msgserver.IMsgServer,
	caCert *x509.Certificate) *AuthnUser {

	svc := &AuthnUser{
		store:     store,
		msgServer: msgServer,
		caCert:    caCert,
	}
	return svc
}
