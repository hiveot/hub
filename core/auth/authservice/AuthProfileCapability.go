package authservice

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"golang.org/x/exp/slog"
)

// AuthProfileCapability is the capability for clients to view and update their own profile.
// This implements the IClientProfile interface.
//
// This implements the IAuthManageProfile interface.
type AuthProfileCapability struct {
	// Client record persistence
	store auth.IAuthnStore
	// message server for updating authn
	msgServer msgserver.IMsgServer
	// CA certificate for validating cert
	caCert *x509.Certificate
}

// CreateToken creates an authentication token using server.
func (svc *AuthProfileCapability) CreateToken(clientID string) (
	newToken string, err error) {

	return svc.msgServer.CreateToken(clientID)
}

// GetProfile returns a client's profile
func (svc *AuthProfileCapability) GetProfile(clientID string) (profile auth.ClientProfile, err error) {
	clientProfile, err := svc.store.GetProfile(clientID)
	return clientProfile, err
}

// NewToken validates a password and issues an authn token. A public key must be on file.
func (svc *AuthProfileCapability) NewToken(clientID string, password string) (newToken string, err error) {
	clientProfile, err := svc.store.VerifyPassword(clientID, password)
	if err != nil {
		return "", err
	}
	if clientProfile.PubKey == "" {
		return "", fmt.Errorf("no public key on file for '%s'", clientID)
	}
	newToken, err = svc.CreateToken(clientID)
	return newToken, err
}

// notification handler invoked when clients have been updated
// this invokes a reload of server authn
func (svc *AuthProfileCapability) onChange() {
	// wait with applying credential changes as it requires a new login
	go svc.msgServer.ApplyAuth(svc.store.GetAuthClientList())
}

// Refresh issues a new token if the given token is valid
// This returns a refreshed token that can be used to connect to the messaging server
// the old token must be a valid jwt token belonging to the clientID
func (svc *AuthProfileCapability) Refresh(clientID string, oldToken string) (newToken string, err error) {
	// verify the token
	clientProfile, err := svc.store.GetProfile(clientID)
	if err != nil {
		return "", err
	}
	err = svc.msgServer.ValidateToken(
		clientID, clientProfile.PubKey, oldToken, "", "")
	if err != nil {
		return "", fmt.Errorf("error validating oldToken of client %s: %w", clientID, err)
	}
	newToken, err = svc.CreateToken(clientID)
	return newToken, err
}

// UpdateName
func (svc *AuthProfileCapability) UpdateName(clientID string, displayName string) (err error) {
	clientProfile, err := svc.store.GetProfile(clientID)
	clientProfile.DisplayName = displayName
	err = svc.store.Update(clientID, clientProfile)
	// this doesn't affect authentication
	return err
}

func (svc *AuthProfileCapability) UpdatePassword(clientID string, newPassword string) (err error) {
	slog.Info("UpdatePassword", "clientID", clientID)
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

func (svc *AuthProfileCapability) UpdatePubKey(clientID string, newPubKey string) (err error) {
	slog.Info("UpdatePubKey", "clientID", clientID)
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
//func (svc *AuthProfileCapability) ValidateToken(clientID string, oldToken string) (err error) {
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
//func (svc *AuthProfileCapability) ValidateCert(clientID string, clientCertPEM string) error {
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

// NewAuthProfileCapability returns a user profile management capability.
//
//	store holds the authentication client records
//	caCert is an optional CA used to verify certificates. Use nil to not authn using client certs
func NewAuthProfileCapability(
	store auth.IAuthnStore,
	msgServer msgserver.IMsgServer,
	caCert *x509.Certificate) *AuthProfileCapability {

	svc := &AuthProfileCapability{
		store:     store,
		msgServer: msgServer,
		caCert:    caCert,
	}
	return svc
}
