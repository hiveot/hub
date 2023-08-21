package authnservice

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/core/msgserver/natsserver"
)

// UserAuthnService handles authentication user requests
// This implements the IAuthnUser interface.
//
// This implements the IAuthnUser interface.
type UserAuthnService struct {
	// Client record persistence
	store authn.IAuthnStore
	// Tokenizer to use
	tokenizer authn.IAuthnTokenizer
	// message server for updating authn
	msgServer *natsserver.NatsNKeyServer
	// CA certificate for validating cert
	caCert *x509.Certificate
}

// CreateToken creates an authentication token using the external tokenizer or
// the built-in tokenizer.
// This invokes the external tokenizer if provided and falls-back to the built-in
// tokenizer.
func (svc *UserAuthnService) CreateToken(clientID string, clientType string, pubKey string, validitySec int) (newToken string, err error) {
	return svc.tokenizer.CreateToken(clientID, clientType, pubKey, validitySec)
}

// GeneratePassword with upper, lower, numbers and special characters
//func (svc *UserAuthnService) GeneratePassword(length int, useSpecial bool) (password string) {
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
func (svc *UserAuthnService) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	entry, err := svc.store.Get(clientID)
	return entry, err
}

// NewToken validates a password and issues an authn token
func (svc *UserAuthnService) NewToken(clientID string, password string) (newToken string, err error) {
	entry, err := svc.store.VerifyPassword(clientID, password)
	if err != nil {
		return "", err
	}
	if entry.PubKey == "" {
		return "", fmt.Errorf("no public key on file for '%s'", clientID)
	}
	newToken, err = svc.CreateToken(clientID, entry.ClientType, entry.PubKey, entry.ValiditySec)
	return newToken, err
}

// Refresh issues a new token if the given token is valid
// This returns a refreshed token that can be used to connect to the messaging server
// the old token must be a valid jwt token belonging to the clientID
func (svc *UserAuthnService) Refresh(clientID string, oldToken string) (newToken string, err error) {
	// verify the token
	entry, err := svc.store.Get(clientID)
	if err != nil {
		return "", err
	}
	err = svc.tokenizer.ValidateToken(clientID, entry.PubKey, oldToken, "", "")
	if err != nil {
		return "", fmt.Errorf("error validating oldToken of client %s: %w", clientID, err)
	}
	newToken, err = svc.CreateToken(clientID, entry.ClientType, entry.PubKey, entry.ValiditySec)
	return newToken, err
}

// UpdateName
func (svc *UserAuthnService) UpdateName(clientID string, displayName string) (err error) {
	entry, err := svc.store.Get(clientID)
	entry.DisplayName = displayName
	err = svc.store.Update(clientID, entry)
	return err
}

func (svc *UserAuthnService) UpdatePassword(clientID string, newPassword string) (err error) {
	entry, err := svc.GetProfile(clientID)
	if err != nil {
		return err
	}
	err = svc.store.SetPassword(clientID, newPassword)
	if err != nil {
		return err
	}
	_ = entry
	return err
}

func (svc *UserAuthnService) UpdatePubKey(clientID string, newPubKey string) (err error) {
	entry, err := svc.store.Get(clientID)
	if err != nil {
		return err
	}
	entry.PubKey = newPubKey
	err = svc.store.Update(clientID, entry)
	if err != nil {
		return err
	}

	return err
}

// ValidateToken verifies if the token is valid and belongs to the claimed user
//func (svc *UserAuthnService) ValidateToken(clientID string, oldToken string) (err error) {
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
//func (svc *UserAuthnService) ValidateCert(clientID string, clientCertPEM string) error {
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

// NewUserAuthnService returns a user authentication capability.
//
//	store holds the authentication client records
//	tokenizer is an optional alternative implementation of token issue and verification
//	caCert is an optional CA used to verify certificates. Use nil to not authn using client certs
func NewUserAuthnService(
	store authn.IAuthnStore,
	msgServer *natsserver.NatsNKeyServer,
	tokenizer authn.IAuthnTokenizer,
	caCert *x509.Certificate) *UserAuthnService {

	svc := &UserAuthnService{
		store:     store,
		msgServer: msgServer,
		tokenizer: tokenizer,
		caCert:    caCert,
	}
	return svc
}
