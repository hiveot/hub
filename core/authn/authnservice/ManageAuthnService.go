package authnservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"golang.org/x/exp/slog"
)

// ManageAuthnService handles authentication management requests
// This applies the request to the store.
//
// Note: To apply authn to the messaging server, authz has to be set.
// Note: Unfortunately authn and authz are tightly coupled in NATS. To set
// authn, permissions must be known. So for dependencies sake,
// authz will invoke msgServer.ReloadClients(authnStore, authzStore)
type ManageAuthnService struct {
	// clients storage
	store authn.IAuthnStore
}

// AddDevice adds an IoT device and generates an authentication token
// This is handled by the underlying messaging core.
func (svc *ManageAuthnService) AddDevice(
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
	// the token will be applied when authorization (group membership) is set
	return pubKey, err
}

// AddService adds or updates a service
func (svc *ManageAuthnService) AddService(
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
	// the token will be applied when authorization (group membership) is set
	token = pubKey
	return token, err
}

// AddUser adds a new user for password authentication
// If a public key is provided a signed token will be returned
func (svc *ManageAuthnService) AddUser(
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
	// the token will be applied when authorization (group membership) is set
	token = pubKey
	return token, err
}

// GetClientProfile returns a client's profile
func (svc *ManageAuthnService) GetClientProfile(clientID string) (profile authn.ClientProfile, err error) {
	entry, err := svc.store.Get(clientID)
	return entry, err
}

func (svc *ManageAuthnService) GetCount() (int, error) {
	return svc.store.Count(), nil
}

// ListClients provide a list of known clients and their info.
func (svc *ManageAuthnService) ListClients() (profiles []authn.ClientProfile, err error) {
	profiles, err = svc.store.List()
	return profiles, err
}

// RemoveClient removes a client and disables authentication
func (svc *ManageAuthnService) RemoveClient(clientID string) (err error) {
	err = svc.store.Remove(clientID)
	return err
}

func (svc *ManageAuthnService) UpdateClient(clientID string, prof authn.ClientProfile) (err error) {
	err = svc.store.Update(clientID, prof)
	return err
}

// NewManageAuthnService creates the capability to manage authentication clients
//
//	store for storing clients
func NewManageAuthnService(
	store authn.IAuthnStore) *ManageAuthnService {
	svc := &ManageAuthnService{
		store: store,
	}
	return svc
}
