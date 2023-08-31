package authservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"golang.org/x/exp/slog"
	"time"
)

// AuthClientsCapability handles management of devices,users and service clients
// This applies the request to the store.
//
// Note: To apply authn to the messaging server, authz has to be set.
// Note: Unfortunately authn and authz are tightly coupled in NATS. To set
// authn, permissions must be known. So for dependencies sake.
type AuthClientsCapability struct {
	// clients storage
	store auth.IAuthnStore
	// message server to apply changes to
	msgServer msgserver.IMsgServer
}

// AddDevice adds an IoT device and generates an authentication token
// This is handled by the underlying messaging core.
func (svc *AuthClientsCapability) AddDevice(
	deviceID string, name string, pubKey string, tokenValidity time.Duration) (token string, err error) {
	slog.Info("AddDevice",
		slog.String("deviceID", deviceID),
		slog.String("name", name),
		slog.String("pubKey", pubKey))

	if deviceID == "" {
		return "", fmt.Errorf("AddDevice: missing device ID")
	}
	// store/update device.
	err = svc.store.Add(deviceID, auth.ClientProfile{
		ClientID:      deviceID,
		ClientType:    auth.ClientTypeDevice,
		DisplayName:   name,
		PubKey:        pubKey,
		Role:          auth.ClientRoleNone,
		TokenValidity: tokenValidity,
	})
	if err != nil {
		return "", err
	}
	// the token will be applied when authorization (group membership) is set
	svc.onChange()
	return pubKey, err
}

// AddService adds or updates a service with the admin role
func (svc *AuthClientsCapability) AddService(
	serviceID string, name string, pubKey string, tokenValidity time.Duration) (token string, err error) {
	slog.Info("AddService",
		slog.String("serviceID", serviceID),
		slog.String("name", name),
		slog.String("pubKey", pubKey))

	if serviceID == "" {
		return "", fmt.Errorf("missing service ID")
	}
	err = svc.store.Add(serviceID, auth.ClientProfile{
		ClientID:      serviceID,
		ClientType:    auth.ClientTypeService,
		DisplayName:   name,
		PubKey:        pubKey,
		Role:          auth.ClientRoleAdmin,
		TokenValidity: tokenValidity,
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
func (svc *AuthClientsCapability) AddUser(
	userID string, userName string, password string, pubKey string, role string) (token string, err error) {

	slog.Info("AddUser",
		slog.String("userID", userID),
		slog.String("userName", userName),
		slog.String("pubKey", pubKey),
		slog.String("role", role))

	if userID == "" {
		return "", fmt.Errorf("missing user ID")
	}
	err = svc.store.Add(userID, auth.ClientProfile{
		ClientID:      userID,
		ClientType:    auth.ClientTypeUser,
		DisplayName:   userName,
		PubKey:        pubKey,
		Role:          role,
		TokenValidity: auth.DefaultUserTokenValidity,
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

func (svc *AuthClientsCapability) GetCount() (int, error) {
	return svc.store.Count(), nil
}

func (svc *AuthClientsCapability) GetAuthClientList() []msgserver.AuthClient {
	return svc.store.GetAuthClientList()
}

// GetProfile returns a client's profile
func (svc *AuthClientsCapability) GetProfile(clientID string) (profile auth.ClientProfile, err error) {
	entry, err := svc.store.GetProfile(clientID)
	return entry, err
}

// GetProfiles provide a list of known clients and their info.
func (svc *AuthClientsCapability) GetProfiles() (profiles []auth.ClientProfile, err error) {
	profiles, err = svc.store.GetProfiles()
	return profiles, err
}

// GetEntries provide a list of known clients and their info including bcrypted passwords
func (svc *AuthClientsCapability) GetEntries() (entries []auth.AuthnEntry) {
	return svc.store.GetEntries()
}

// notification handler invoked when clients have been added, removed or updated
// this invokes a reload of server authn
func (svc *AuthClientsCapability) onChange() {
	entries := svc.store.GetEntries()
	clients := make([]msgserver.AuthClient, 0, len(entries))
	for _, e := range entries {
		clients = append(clients, msgserver.AuthClient{
			ClientID:     e.ClientID,
			ClientType:   e.ClientType,
			PubKey:       e.PubKey,
			PasswordHash: e.PasswordHash,
			Role:         e.Role,
		})
	}
	_ = svc.msgServer.ApplyAuth(clients)
}

// RemoveClient removes a client and disables authentication
func (svc *AuthClientsCapability) RemoveClient(clientID string) (err error) {
	err = svc.store.Remove(clientID)
	svc.onChange()
	return err
}

func (svc *AuthClientsCapability) UpdateClient(clientID string, prof auth.ClientProfile) (err error) {
	err = svc.store.Update(clientID, prof)
	return err
}

// NewAuthClientsCapability creates the capability to manage authentication clients
//
//	store for storing clients
func NewAuthClientsCapability(store auth.IAuthnStore, msgServer msgserver.IMsgServer) *AuthClientsCapability {
	svc := &AuthClientsCapability{
		store:     store,
		msgServer: msgServer,
	}
	return svc
}
