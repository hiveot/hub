package authnstore

import (
	"encoding/json"
	"errors"
	"fmt"
	authn2 "github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/config"
	"log/slog"
	"os"
	"path"
	"sync"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/fsnotify/fsnotify"
	"golang.org/x/crypto/bcrypt"

	"github.com/hiveot/hub/lib/watcher"
)

// AuthnFileStore stores client data, including users, devices and services.
// User passwords are stored using ARGON2id hash
// It includes a file watcher to automatically reload on update.
type AuthnFileStore struct {
	entries           map[string]api.AuthnEntry
	storePath         string
	hashAlgo          string // hashing algorithm PWHASH_ARGON2id
	minPasswordLength int
	watcher           *fsnotify.Watcher
	mutex             sync.RWMutex
}

// Add a new client.
// clientID, clientType are required, the rest is optional
func (store *AuthnFileStore) Add(clientID string, profile authn2.ClientProfile) error {

	store.mutex.Lock()
	defer store.mutex.Unlock()

	entry, found := store.entries[clientID]
	if clientID == "" || clientID != profile.ClientID {
		return fmt.Errorf("Add: missing clientID")
	}
	if profile.ClientType != authn2.ClientTypeAgent &&
		profile.ClientType != authn2.ClientTypeConsumer &&
		profile.ClientType != authn2.ClientTypeService {
		return fmt.Errorf("Add: invalid clientType '%s' for client '%s'",
			profile.ClientType, clientID)
	}

	if !found {
		slog.Info("Add: New client " + clientID)
		entry = api.AuthnEntry{ClientProfile: profile}
	} else {
		slog.Info("Add: Updating existing client", slog.String("clientID", clientID))
		entry.ClientProfile = profile
	}
	entry.Updated = time.Now().UnixMilli()

	store.entries[clientID] = entry

	err := store.save()
	return err
}

// Close the store
func (store *AuthnFileStore) Close() {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.watcher != nil {
		_ = store.watcher.Close()
		store.watcher = nil
	}
}

// Count nr of entries in the store
func (store *AuthnFileStore) Count() int {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	return len(store.entries)
}

// GetProfile returns the client's profile
func (store *AuthnFileStore) GetProfile(
	clientID string) (profile authn2.ClientProfile, err error) {

	store.mutex.RLock()
	defer store.mutex.RUnlock()
	// user must exist
	entry, found := store.entries[clientID]
	if !found {
		err = fmt.Errorf("clientID '%s' does not exist", clientID)
	}
	return entry.ClientProfile, err
}

// GetProfiles returns a list of all client profiles in the store
func (store *AuthnFileStore) GetProfiles() (profiles []authn2.ClientProfile, err error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	profiles = make([]authn2.ClientProfile, 0, len(store.entries))
	for _, entry := range store.entries {
		profiles = append(profiles, entry.ClientProfile)
	}
	return profiles, nil
}

// GetRole returns the client's stored role
func (store *AuthnFileStore) GetRole(clientID string) (role string, err error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	// user must exist
	entry, found := store.entries[clientID]
	if !found {
		err = fmt.Errorf("clientID '%s' does not exist", clientID)
		return "", err
	}
	return entry.Role, nil
}

// GetEntries returns a list of all profiles with their hashed passwords
func (store *AuthnFileStore) GetEntries() (entries []api.AuthnEntry) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	entries = make([]api.AuthnEntry, 0, len(store.entries))
	for _, entry := range store.entries {
		entries = append(entries, entry)
	}
	return entries
}

// Open the store
// This reads the password file and subscribes to file changes
func (store *AuthnFileStore) Open() (err error) {
	if store.watcher != nil {
		err = fmt.Errorf("password file store '%s' is already open", store.storePath)
	}
	if err == nil {
		err = store.Reload()
	}
	if err == nil {
		store.watcher, err = watcher.WatchFile(store.storePath, store.Reload)
	}
	if err != nil {
		err = fmt.Errorf("NewSession failed %w", err)
	}
	return err
}

// Reload the password store from file and subscribe to file changes
//
// If the file does not exist, it will be created.
// Returns an error if the file could not be opened/created.
func (store *AuthnFileStore) Reload() error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	entries := make(map[string]api.AuthnEntry)
	dataBytes, err := os.ReadFile(store.storePath)
	if errors.Is(err, os.ErrNotExist) {
		err = store.save()
	} else if err != nil {
		err = fmt.Errorf("error reading password file: %w", err)
		return err
	} else if len(dataBytes) == 0 {
		// nothing to do
	} else {

		err = json.Unmarshal(dataBytes, &entries)
		if err != nil {
			err := fmt.Errorf("error while parsing password file: %w", err)
			return err
		}
		store.entries = entries
	}
	return err
}

// Remove a client from the store
func (store *AuthnFileStore) Remove(clientID string) (err error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	_, found := store.entries[clientID]
	if found {
		delete(store.entries, clientID)
	}
	err = store.save()
	return err
}

// save the password data to file
// if the storage folder doesn't exist it will be created
// not concurrent save
func (store *AuthnFileStore) save() error {

	folder := path.Dir(store.storePath)
	// ensure the location exists
	err := os.MkdirAll(folder, 0700)
	if err != nil {
		return err
	}
	tmpPath, err := WritePasswordsToTempFile(folder, store.entries)
	if err != nil {
		err = fmt.Errorf("writing password file to temp failed: %w", err)
		return err
	}

	err = os.Rename(tmpPath, store.storePath)
	if err != nil {
		err = fmt.Errorf("rename to password file failed: %w", err)
		return err
	}
	return err
}

// SetPassword generates and stores the user's password hash.
//
// The hash used is argon2id or bcrypt based on the 'hashAlgo' setting.
// bcrypt limits max password length to 72 bytes.
func (store *AuthnFileStore) SetPassword(loginID string, password string) (err error) {
	var hash string
	if len(password) < store.minPasswordLength {
		return fmt.Errorf("password too short (%d chars)", len(password))
	}
	if store.hashAlgo == config.PWHASH_ARGON2id {
		// TODO: tweak to something reasonable and test timing. default of 64MB is not suitable for small systems
		params := argon2id.DefaultParams
		params.Memory = 16 * 1024
		params.Iterations = 2
		params.Parallelism = 4 // what happens with fewer cores?
		hash, err = argon2id.CreateHash(password, params)
	} else if store.hashAlgo == config.PWHASH_BCRYPT {
		hashBytes, err2 := bcrypt.GenerateFromPassword([]byte(password), 0)
		err = err2
		hash = string(hashBytes)
	} else {
		err = fmt.Errorf("unknown password hash: %s", store.hashAlgo)
	}
	if err != nil {
		return err
	}
	return store.SetPasswordHash(loginID, hash)
}

// SetPasswordHash adds/updates the password hash for the given login ID
// Intended for clients to update their own password
func (store *AuthnFileStore) SetPasswordHash(loginID string, hash string) (err error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	entry, found := store.entries[loginID]
	if !found {
		return fmt.Errorf("Client '%s' not found", loginID)
	}
	entry.PasswordHash = hash
	entry.Updated = time.Now().UnixMilli()
	store.entries[loginID] = entry

	err = store.save()
	return err
}

// SetRole changes the client's default role
func (store *AuthnFileStore) SetRole(clientID string, role string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	entry, found := store.entries[clientID]
	if !found {
		return fmt.Errorf("SetRole: Client '%s' not found", clientID)
	}
	entry.Role = role
	store.entries[clientID] = entry
	err := store.save()
	return err
}

// UpdateProfile updates the client profile
// The senderID is the connection ID of the sender.
// Authorization should only allow admin level
func (store *AuthnFileStore) UpdateProfile(senderID string, profile authn2.ClientProfile) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	// TODO: verify sender exi
	entry, found := store.entries[senderID]
	if !found {
		return fmt.Errorf("UpdateProfile: SenderID='%s' not found", senderID)
	}

	entry, found = store.entries[profile.ClientID]
	if !found {
		return fmt.Errorf("UpdateProfile: SenderID='%s'; Client '%s' not found",
			senderID, profile.ClientID)
	}
	if profile.ClientType != "" {
		entry.ClientType = profile.ClientType
	}
	if profile.DisplayName != "" {
		entry.DisplayName = profile.DisplayName
	}
	if profile.PubKey != "" {
		entry.PubKey = profile.PubKey
	}
	entry.Updated = time.Now().UnixMilli()
	store.entries[profile.ClientID] = entry

	err := store.save()
	return err
}

// VerifyPassword verifies the given password with the stored hash
// This returns the matching user's entry or an error if the password doesn't match
func (store *AuthnFileStore) VerifyPassword(
	loginID, password string) (profile authn2.ClientProfile, err error) {
	isValid := false
	store.mutex.Lock()
	defer store.mutex.Unlock()

	entry, found := store.entries[loginID]
	if !found {
		// unknown user
		isValid = false
	} else if store.hashAlgo == config.PWHASH_ARGON2id {
		isValid, _ = argon2id.ComparePasswordAndHash(password, entry.PasswordHash)
	} else if store.hashAlgo == config.PWHASH_BCRYPT {
		err := bcrypt.CompareHashAndPassword([]byte(entry.PasswordHash), []byte(password))
		isValid = err == nil
	}
	if !isValid {
		return profile, fmt.Errorf("invalid login as '%s'", loginID)
	}
	profile = entry.ClientProfile
	return profile, nil
}

// WritePasswordsToTempFile write the given entries to temp file in the given folder
// This returns the name of the new temp file.
func WritePasswordsToTempFile(
	folder string, entries map[string]api.AuthnEntry) (tempFileName string, err error) {

	file, err := os.CreateTemp(folder, "hub-pwfilestore")

	// file, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		err := fmt.Errorf("failed open temp password file: %s", err)
		return "", err
	}
	tempFileName = file.Name()

	defer file.Close()
	pwData, err := json.Marshal(entries)
	if err == nil {
		_, err = file.Write(pwData)
	}

	return tempFileName, err
}

// NewAuthnFileStore creates a new instance of a file based identity store.
// Call NewSession/Release to start/stop using this store.
// Note: this store is intended for one writer and many readers.
// Multiple concurrent writes are not supported and might lead to one write being ignored.
//
//	filepath location of the file store. See also DefaultPasswordFile for the recommended name
//	hashAlgo PWHASH_ARGON2id (default) or PWHASH_BCRYPT
func NewAuthnFileStore(filepath string, hashAlgo string) *AuthnFileStore {
	if hashAlgo == "" {
		hashAlgo = config.PWHASH_ARGON2id
	}
	if hashAlgo != config.PWHASH_ARGON2id && hashAlgo != config.PWHASH_BCRYPT {
		slog.Error("unknown hash algorithm. Falling back to argon2id", "hashAlgo", hashAlgo)
	}
	store := &AuthnFileStore{
		storePath:         filepath,
		hashAlgo:          hashAlgo,
		minPasswordLength: 5,
		entries:           make(map[string]api.AuthnEntry),
	}
	return store
}
