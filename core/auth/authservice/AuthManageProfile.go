package authservice

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
)

// AuthManageProfile is the capability for clients to view and update their own profile.
// This implements the IManageProfile interface.
//
// This implements the IAuthManageProfile interface.
type AuthManageProfile struct {
	// Client record persistence
	store auth.IAuthnStore
	// hub client for subscribing to requests
	hc hubclient.IHubClient
	// action subscription
	actionSub hubclient.ISubscription

	// message server for updating authn
	msgServer msgserver.IMsgServer
	// CA certificate for validating cert
	caCert *x509.Certificate
}

// GetProfile returns a client's profile
func (svc *AuthManageProfile) GetProfile(clientID string) (profile auth.ClientProfile, err error) {
	clientProfile, err := svc.store.GetProfile(clientID)
	return clientProfile, err
}

// HandleActions unmarshal and invoke requests published by hub clients
func (svc *AuthManageProfile) HandleActions(action *hubclient.ActionRequest) error {
	if action.ClientID == "" {
		return fmt.Errorf("missing clientID in action request", "deviceID", action.DeviceID, "thingID", action.ThingID)
	}
	slog.Info("handleClientActions", slog.String("actionID", action.ActionID))
	switch action.ActionID {
	case auth.GetProfileAction:
		// use the current client
		profile, err := svc.GetProfile(action.ClientID)
		if err == nil {
			resp := auth.GetProfileResp{Profile: profile}
			reply, _ := ser.Marshal(&resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case auth.NewTokenAction:
		req := &auth.NewTokenReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		newToken, err := svc.NewToken(action.ClientID, req.Password)
		if err == nil {
			resp := auth.NewTokenResp{Token: newToken}
			reply, _ := ser.Marshal(resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case auth.RefreshAction:
		req := &auth.RefreshReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		newToken, err := svc.Refresh(action.ClientID, req.OldToken)
		if err == nil {
			resp := auth.RefreshResp{NewToken: newToken}
			reply, _ := ser.Marshal(resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case auth.UpdateNameAction:
		req := &auth.UpdateNameReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.UpdateName(action.ClientID, req.NewName)
		if err == nil {
			err = action.SendAck()
		}
		return err
	case auth.UpdatePasswordAction:
		req := &auth.UpdatePasswordReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.UpdatePassword(action.ClientID, req.NewPassword)
		if err == nil {
			err = action.SendAck()
		}
		return err
	case auth.UpdatePubKeyAction:
		req := &auth.UpdatePubKeyReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.UpdatePubKey(action.ClientID, req.NewPubKey)
		if err == nil {
			err = action.SendAck()
		}
		return err
	default:
		return fmt.Errorf("Unknown user action '%s' for client '%s'", action.ActionID, action.ClientID)
	}
}

// NewToken validates a password and issues an authn token. A public key must be on file.
func (svc *AuthManageProfile) NewToken(clientID string, password string) (newToken string, err error) {
	_, err = svc.store.VerifyPassword(clientID, password)
	if err != nil {
		return "", err
	}
	newToken, err = svc.msgServer.CreateToken(clientID)
	return newToken, err
}

// notification handler invoked when clients have been updated
// this invokes a reload of server authn
func (svc *AuthManageProfile) onChange() {
	// wait with applying credential changes to allow a response to be send
	go svc.msgServer.ApplyAuth(svc.store.GetAuthClientList())
}

// Refresh issues a new token if the given token is valid
// This returns a refreshed token that can be used to connect to the messaging server
// the old token must be a valid jwt token belonging to the clientID
func (svc *AuthManageProfile) Refresh(clientID string, oldToken string) (newToken string, err error) {
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
	newToken, err = svc.msgServer.CreateToken(clientID)
	return newToken, err
}

// Start subscribes to the actions for management and client capabilities
// Register the binding subscription using the given connection
func (svc *AuthManageProfile) Start() (err error) {
	if svc.hc != nil {
		svc.actionSub, _ = svc.hc.SubServiceActions(
			auth.AuthProfileCapability, svc.HandleActions)
	}
	return err
}

// Stop removes subscriptions
func (svc *AuthManageProfile) Stop() {
	if svc.actionSub != nil {
		svc.actionSub.Unsubscribe()
		svc.actionSub = nil
	}
}

// UpdateName
func (svc *AuthManageProfile) UpdateName(clientID string, displayName string) (err error) {
	clientProfile, err := svc.store.GetProfile(clientID)
	clientProfile.DisplayName = displayName
	err = svc.store.Update(clientID, clientProfile)
	// this doesn't affect authentication
	return err
}

func (svc *AuthManageProfile) UpdatePassword(clientID string, newPassword string) (err error) {
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

func (svc *AuthManageProfile) UpdatePubKey(clientID string, newPubKey string) (err error) {
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
//func (svc *AuthManageProfile) ValidateToken(clientID string, oldToken string) (err error) {
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
//func (svc *AuthManageProfile) ValidateCert(clientID string, clientCertPEM string) error {
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

// NewAuthManageProfile returns a user profile management capability.
//
//	store holds the authentication client records
//	caCert is an optional CA used to verify certificates. Use nil to not authn using client certs
func NewAuthManageProfile(
	store auth.IAuthnStore,
	caCert *x509.Certificate,
	hc hubclient.IHubClient,
	msgServer msgserver.IMsgServer,
) *AuthManageProfile {

	svc := &AuthManageProfile{
		store:     store,
		hc:        hc,
		msgServer: msgServer,
		caCert:    caCert,
	}
	return svc
}
