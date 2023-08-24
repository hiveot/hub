package authnservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/msgserver"
	"golang.org/x/exp/slog"
)

// AuthnManage handles authentication management requests
// This applies the request to the store.
//
// Note: To apply authn to the messaging server, authz has to be set.
// Note: Unfortunately authn and authz are tightly coupled in NATS. To set
// authn, permissions must be known. So for dependencies sake.
type AuthnManage struct {
	// clients storage
	store authn.IAuthnStore
	// message server to apply changes to
	msgServer msgserver.IMsgServer
}

// AddDevice adds an IoT device and generates an authentication token
// This is handled by the underlying messaging core.
func (svc *AuthnManage) AddDevice(
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
	svc.onChange()
	return pubKey, err
}

// AddService adds or updates a service
func (svc *AuthnManage) AddService(
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
	svc.onChange()
	return token, err
}

// AddUser adds a new user for password authentication
// If a public key is provided a signed token will be returned
func (svc *AuthnManage) AddUser(
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
	svc.onChange()
	return token, err
}

func (svc *AuthnManage) GetCount() (int, error) {
	return svc.store.Count(), nil
}

// GetProfile returns a client's profile
func (svc *AuthnManage) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	entry, err := svc.store.GetProfile(clientID)
	return entry, err
}

// GetProfiles provide a list of known clients and their info.
func (svc *AuthnManage) GetProfiles() (profiles []authn.ClientProfile, err error) {
	profiles, err = svc.store.GetProfiles()
	return profiles, err
}

// GetEntries provide a list of known clients and their info including bcrypted passwords
func (svc *AuthnManage) GetEntries() (entries []authn.AuthnEntry) {
	return svc.store.GetEntries()
}

// notification handler invoked when clients have been added, removed or updated
// this invokes a reload of server authn
func (svc *AuthnManage) onChange() {
	_ = svc.msgServer.ApplyAuthn(svc.store.GetEntries())
}

// RemoveClient removes a client and disables authentication
func (svc *AuthnManage) RemoveClient(clientID string) (err error) {
	err = svc.store.Remove(clientID)
	svc.onChange()
	return err
}

func (svc *AuthnManage) UpdateClient(clientID string, prof authn.ClientProfile) (err error) {
	err = svc.store.Update(clientID, prof)
	return err
}

// NewAuthnManage creates the capability to manage authentication clients
//
//	store for storing clients
func NewAuthnManage(store authn.IAuthnStore, msgServer msgserver.IMsgServer) *AuthnManage {
	svc := &AuthnManage{
		store:     store,
		msgServer: msgServer,
	}
	return svc
}
