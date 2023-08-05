package authnservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/core/authn/authnstore"
	"golang.org/x/exp/slog"
)

// AuthnManageService handles authentication management and user requests
//
// This applies the request to the store and the underlying core service.
type AuthnManageService struct {
	// clients storage
	store authnstore.IAuthnStore
	// tokenizer for handling tokens
	tokenizer authn.IAuthnTokenizer
}

// AddDevice adds an IoT device and generates an authentication token
// This is handled by the underlying messaging core.
func (svc *AuthnManageService) AddDevice(
	deviceID string, name string, pubKey string, validitySec int) (token string, err error) {

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
	// create the token for the underlying messaging core
	if pubKey != "" {
		token, err = svc.tokenizer.CreateToken(deviceID, authn.ClientTypeDevice, pubKey, validitySec)
		if err != nil {
			return token, fmt.Errorf("device '%s' added, but: %w", deviceID, err)
		}
	}
	return token, err
}

// AddService adds or updates a service
func (svc *AuthnManageService) AddService(
	serviceID string, name string, pubKey string, validitySec int) (token string, err error) {

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
	// create the token for the underlying messaging core
	if pubKey != "" {
		token, err = svc.tokenizer.CreateToken(serviceID, authn.ClientTypeService, pubKey, validitySec)
		if err != nil {
			return token, fmt.Errorf("service '%s' added, but: %w", serviceID, err)
		}
	}
	return token, err
}

// AddUser adds a new user for password authentication
// If a public key is provided a signed token will be returned
func (svc *AuthnManageService) AddUser(
	userID string, userName string, password string, pubKey string) (token string, err error) {
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
		return "", fmt.Errorf("user with clientID '%s' already exists", userID)
	}
	if password != "" {
		err = svc.store.SetPassword(userID, password)
		if err != nil {
			err = fmt.Errorf("AddUser: user '%s' added, but: %w. Continuing", userID, err)
			slog.Error(err.Error())
		}
	}
	// create the token for the underlying messaging core
	if pubKey != "" {
		token, err = svc.tokenizer.CreateToken(userID, authn.ClientTypeUser, pubKey, authn.DefaultUserTokenValiditySec)
		if err != nil {
			err = fmt.Errorf("AddUser: user '%s' added, but: %w. Continuing", userID, err)
			slog.Error(err.Error())
		}
	}
	return token, err
}

// GetClientProfile returns a client's profile
func (svc *AuthnManageService) GetClientProfile(clientID string) (profile authn.ClientProfile, err error) {
	entry, err := svc.store.Get(clientID)
	return entry, err
}

func (svc *AuthnManageService) GetCount() (int, error) {
	return svc.store.Count(), nil
}

// ListClients provide a list of known clients and their info.
func (svc *AuthnManageService) ListClients() (profiles []authn.ClientProfile, err error) {
	profiles, err = svc.store.List()
	return profiles, err
}

// RemoveClient removes a client and disables authentication
func (svc *AuthnManageService) RemoveClient(clientID string) (err error) {
	err = svc.store.Remove(clientID)
	return err
}

func (svc *AuthnManageService) UpdateClient(clientID string, prof authn.ClientProfile) (err error) {
	err = svc.store.Update(clientID, prof)
	return err
}

// NewAuthnManageService creates the service to manage authentication clients
//
//	store for storing clients
//	tokenizer to generate tokens for authentication with the underlying messaging service
func NewAuthnManageService(store authnstore.IAuthnStore, tokenizer authn.IAuthnTokenizer) *AuthnManageService {
	svc := &AuthnManageService{
		store:     store,
		tokenizer: tokenizer,
	}
	return svc
}
