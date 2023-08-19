package authnservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/core/authn/authnstore"
	"github.com/hiveot/hub/core/msgserver/natsserver"
	"golang.org/x/exp/slog"
)

// AuthnManageThing handles authentication management and user requests
//
// This applies the request to the store and the underlying core service.
type AuthnManageThing struct {
	// clients storage
	store authnstore.IAuthnStore
	// tokenizer for handling tokens
	tokenizer authn.IAuthnTokenizer
	// server for updating nkeys and users
	msgServer *natsserver.NatsNKeyServer
}

// AddDevice adds an IoT device and generates an authentication token
// This is handled by the underlying messaging core.
func (svc *AuthnManageThing) AddDevice(
	deviceID string, name string, pubKey string, validitySec int) (token string, err error) {
	slog.Info("AddDevice",
		slog.String("deviceID", deviceID),
		slog.String("name", name),
		slog.String("pubKey", pubKey))

	if deviceID == "" {
		return "", fmt.Errorf("AddDevice: missing device ID")
	}
	// store/update device.
	err = svc.store.Add(deviceID, authn.ClientProfile{
		ClientID:    deviceID,
		ClientType:  authn.ClientTypeDevice,
		DisplayName: name,
		PubKey:      pubKey,
		ValiditySec: validitySec,
	})
	if err != nil {
		return "", err
	}
	// apply the token to the underlying messaging core
	if pubKey != "" {
		//	token, err = svc.tokenizer.CreateToken(deviceID, authn.ClientTypeDevice, pubKey, validitySec)
		//	if err != nil {
		//		return token, fmt.Errorf("device '%s' added, but: %w", deviceID, err)
		//	}
		err = svc.msgServer.AddDevice(deviceID, pubKey)
	}
	return pubKey, err
}

// AddService adds or updates a service
func (svc *AuthnManageThing) AddService(
	serviceID string, name string, pubKey string, validitySec int) (token string, err error) {
	slog.Info("AddService",
		slog.String("serviceID", serviceID),
		slog.String("name", name),
		slog.String("pubKey", pubKey))

	if serviceID == "" {
		return "", fmt.Errorf("missing service ID")
	}
	err = svc.store.Add(serviceID, authn.ClientProfile{
		ClientID:    serviceID,
		ClientType:  authn.ClientTypeService,
		DisplayName: name,
		PubKey:      pubKey,
		ValiditySec: validitySec,
	})
	if err != nil {
		return "", err
	}
	// apply the token for the underlying messaging core
	if pubKey != "" {
		//	token, err = svc.tokenizer.CreateToken(serviceID, authn.ClientTypeService, pubKey, validitySec)
		//	if err != nil {
		//		return token, fmt.Errorf("service '%s' added, but: %w", serviceID, err)
		//	}
		err = svc.msgServer.AddService(serviceID, pubKey)
	}
	token = pubKey
	return token, err
}

// AddUser adds a new user for password authentication
// If a public key is provided a signed token will be returned
func (svc *AuthnManageThing) AddUser(
	userID string, userName string, password string, pubKey string) (token string, err error) {

	slog.Info("AddUser",
		slog.String("userID", userID),
		slog.String("userName", userName),
		slog.String("pubKey", pubKey))

	if userID == "" {
		return "", fmt.Errorf("missing user ID")
	}
	err = svc.store.Add(userID, authn.ClientProfile{
		ClientID:    userID,
		ClientType:  authn.ClientTypeUser,
		DisplayName: userName,
		PubKey:      pubKey,
	})
	if err != nil {
		return "", err
	}
	if password != "" {
		err = svc.store.SetPassword(userID, password)
		if err != nil {
			err = fmt.Errorf("AddUser: user '%s' added, but: %w. Continuing", userID, err)
			slog.Error(err.Error())
		}
	}
	// apply the token for the underlying messaging server
	//if pubKey != "" {
	//token, err = svc.tokenizer.CreateToken(userID, authn.ClientTypeUser, pubKey, authn.DefaultUserTokenValiditySec)
	//if err != nil {
	//	err = fmt.Errorf("AddUser: user '%s' added, but: '%w'... continuing... ", userID, err)
	//	slog.Error(err.Error())
	//}
	token = pubKey
	//}
	err = svc.msgServer.AddUser(userID, password, pubKey)
	return token, err
}

// GetClientProfile returns a client's profile
func (svc *AuthnManageThing) GetClientProfile(clientID string) (profile authn.ClientProfile, err error) {
	entry, err := svc.store.Get(clientID)
	return entry, err
}

func (svc *AuthnManageThing) GetCount() (int, error) {
	return svc.store.Count(), nil
}

// ListClients provide a list of known clients and their info.
func (svc *AuthnManageThing) ListClients() (profiles []authn.ClientProfile, err error) {
	profiles, err = svc.store.List()
	return profiles, err
}

// RemoveClient removes a client and disables authentication
func (svc *AuthnManageThing) RemoveClient(clientID string) (err error) {
	err = svc.store.Remove(clientID)
	return err
}

func (svc *AuthnManageThing) UpdateClient(clientID string, prof authn.ClientProfile) (err error) {
	err = svc.store.Update(clientID, prof)
	return err
}

// ApplyToServer applies the users and keys to the server
func (svc *AuthnManageThing) ApplyToServer() error {
	clients, err := svc.store.List()
	if err != nil {
		return err
	}
	// convert to users and keys and change server options
	// FIXME: push clients nkeys and pw to server on startup
	//svc.msgServer.ApplyUsers(clients)
	_ = clients
	return nil
}

// NewAuthnManageThing creates the capability to manage authentication clients
//
//	store for storing clients
//	natsServer to update with keys and users
//	tokenizer to generate tokens for authentication with the underlying messaging service
func NewAuthnManageThing(
	store authnstore.IAuthnStore,
	msgServer *natsserver.NatsNKeyServer,
	tokenizer authn.IAuthnTokenizer) *AuthnManageThing {
	svc := &AuthnManageThing{
		store:     store,
		msgServer: msgServer,
		tokenizer: tokenizer,
	}
	return svc
}
