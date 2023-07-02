package unpwstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/exp/slog"
	"os"
	"path"
	"sync"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/fsnotify/fsnotify"
	"golang.org/x/crypto/bcrypt"

	"github.com/hiveot/hub/lib/watcher"
)

// PasswordFileStore hashes and stores a list of user login name, hashed password and display name
// It includes a file watcher to automatically reload on update.
// The storage format is loginID:hash:userName:updated
type PasswordFileStore struct {
	entries   map[string]PasswordEntry // map [loginID]"loginID:hash:userName:updated:
	storePath string
	hashAlgo  string // hashing algorithm PWHASH_ARGON2id
	watcher   *fsnotify.Watcher
	mutex     sync.RWMutex
}

// Close the store
func (pwStore *PasswordFileStore) Close() {
	slog.Info("PasswordFileStore.Release")
	if pwStore.watcher != nil {
		_ = pwStore.watcher.Close()
		pwStore.watcher = nil
	}
}

// Count nr of entries in the store
func (pwStore *PasswordFileStore) Count() int {
	pwStore.mutex.RLock()
	defer pwStore.mutex.RUnlock()

	return len(pwStore.entries)
}

// Exists returns if loginID already exists
func (pwStore *PasswordFileStore) Exists(loginID string) bool {
	pwStore.mutex.RLock()
	defer pwStore.mutex.RUnlock()

	_, found := pwStore.entries[loginID]
	return found
}

// GetEntry user the user info of the loginID
func (pwStore *PasswordFileStore) GetEntry(loginID string) (entry PasswordEntry, err error) {
	pwStore.mutex.RLock()
	defer pwStore.mutex.RUnlock()
	// user must exist
	entry, found := pwStore.entries[loginID]
	if !found {
		err = fmt.Errorf("loginID '%s' does not exist", loginID)
	}
	return entry, err
}

// List returns a list of users in the store
func (pwStore *PasswordFileStore) List() (entries []PasswordEntry, err error) {

	for _, entry := range pwStore.entries {
		entries = append(entries, entry)
	}
	return entries, nil
}

// Open the store
// This reads the password file and subscribes to file changes
func (pwStore *PasswordFileStore) Open() (err error) {
	if pwStore.watcher != nil {
		err = fmt.Errorf("password file store '%s' is already open", pwStore.storePath)
	}
	if err == nil {
		err = pwStore.Reload()
	}
	if err == nil {
		pwStore.watcher, err = watcher.WatchFile(pwStore.storePath, pwStore.Reload)
	}
	if err != nil {
		slog.Error("Open failed", "err", err)
	}
	return err
}

// Reload the password store from file and subscribe to file changes
//
//	File format:  <loginID>:bcrypt(passwd):<username>:updated
//
// If the file does not exist, it will be created.
// Returns an error if the file could not be opened/created.
func (pwStore *PasswordFileStore) Reload() error {
	slog.Info("Reloading passwords", "file", pwStore.storePath)
	pwStore.mutex.Lock()
	defer pwStore.mutex.Unlock()

	entries := make(map[string]PasswordEntry)
	dataBytes, err := os.ReadFile(pwStore.storePath)
	if errors.Is(err, os.ErrNotExist) {
		slog.Info("password file doesn't yet exist. Creating empty file for the watcher.")
		err = pwStore.save()
	} else if err != nil {
		err = fmt.Errorf("error reading password file: %s", err)
		slog.Error(err.Error())
	} else if len(dataBytes) == 0 {
		slog.Info("password file exists but is empty", "filename", pwStore.storePath)
		// nothing to do
	} else {

		err = json.Unmarshal(dataBytes, &entries)
		if err != nil {
			err := fmt.Errorf("error while parsing password file: %s", err)
			slog.Error(err.Error())
			return err
		}
		pwStore.entries = entries
		slog.Info("Reload: loaded passwords", "N", len(entries))
	}
	return err
}

// Remove a user from the store
func (pwStore *PasswordFileStore) Remove(loginID string) (err error) {
	pwStore.mutex.Lock()
	defer pwStore.mutex.Unlock()

	_, found := pwStore.entries[loginID]
	if found {
		delete(pwStore.entries, loginID)
	}
	err = pwStore.save()
	return err
}

// save the password data to file
// if the storage folder doesn't exist it will be created
// not concurrent save
func (pwStore *PasswordFileStore) save() error {

	folder := path.Dir(pwStore.storePath)
	// ensure the location exists
	err := os.MkdirAll(folder, 0700)
	if err != nil {
		return err
	}
	tmpPath, err := WritePasswordsToTempFile(folder, pwStore.entries)
	if err != nil {
		err = fmt.Errorf("writing password file to temp failed: %s", err)
		return err
	}

	err = os.Rename(tmpPath, pwStore.storePath)
	if err != nil {
		err = fmt.Errorf("rename to password file failed: %s", err)
		return err
	}
	return err
}

// SetName updates the display name of a login ID
func (pwStore *PasswordFileStore) SetName(loginID string, newName string) (err error) {
	pwStore.mutex.Lock()
	defer pwStore.mutex.Unlock()

	entry, found := pwStore.entries[loginID]
	if !found {
		entry = PasswordEntry{
			LoginID:  loginID,
			UserName: newName,
		}
	}
	entry.UserName = newName
	entry.Updated = time.Now().Unix()
	pwStore.entries[loginID] = entry

	err = pwStore.save()
	return err
}

// SetPassword generates and stores the user's password hash
func (pwStore *PasswordFileStore) SetPassword(loginID string, password string) (err error) {
	var hash string
	// TODO: tweak to something reasonable and test timing. default of 64MB is not suitable for small systems
	params := argon2id.DefaultParams
	params.Memory = 16 * 1024
	params.Iterations = 2
	params.Parallelism = 4 // what happens with fewer cores?
	if password == "" {
		hash = ""
	} else {
		hash, err = argon2id.CreateHash(password, params)
		if err != nil {
			// when does CreateHash fail?
			return err
		}
	}
	return pwStore.SetPasswordHash(loginID, hash)
}

// SetPasswordHash adds/updates the password hash for the given login ID
// Intended for use by administrators to add a new user or clients to update their password
func (pwStore *PasswordFileStore) SetPasswordHash(loginID string, hash string) (err error) {
	slog.Info("Update password (hash)", "loginID", loginID)
	pwStore.mutex.Lock()
	defer pwStore.mutex.Unlock()

	entry, found := pwStore.entries[loginID]
	if !found {
		entry = PasswordEntry{
			LoginID:      loginID,
			PasswordHash: hash,
		}
	}
	entry.PasswordHash = hash
	entry.Updated = time.Now().Unix()
	pwStore.entries[loginID] = entry

	slog.Info("password hash updated", "loginID", loginID)
	err = pwStore.save()
	return err
}

// VerifyPassword verifies the given password with the stored hash
// This returns the matching user's entry or an error if the password doesn't match
func (pwStore *PasswordFileStore) VerifyPassword(loginID, password string) (PasswordEntry, error) {
	isValid := false
	pwStore.mutex.Lock()
	defer pwStore.mutex.Unlock()

	entry, found := pwStore.entries[loginID]
	if !found {
		// unknown user
		isValid = false
	} else if pwStore.hashAlgo == PWHASH_ARGON2id {
		isValid, _ = argon2id.ComparePasswordAndHash(password, entry.PasswordHash)
	} else if pwStore.hashAlgo == PWHASH_BCRYPT {
		err := bcrypt.CompareHashAndPassword([]byte(entry.PasswordHash), []byte(password))
		isValid = err == nil
	}
	if !isValid {
		return entry, fmt.Errorf("invalid login as '%s'", loginID)
	}
	return entry, nil
}

// WritePasswordsToTempFile write the given entries to temp file in the given folder
// This returns the name of the new temp file.
func WritePasswordsToTempFile(
	folder string, entries map[string]PasswordEntry) (tempFileName string, err error) {

	file, err := os.CreateTemp(folder, "hub-pwfilestore")

	// file, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		err := fmt.Errorf("failed open temp password file: %s", err)
		slog.Error(err.Error())
		return "", err
	}
	tempFileName = file.Name()
	slog.Info("Write passwords to tempfile", "filename", tempFileName)

	defer file.Close()
	pwData, err := json.Marshal(entries)
	if err == nil {
		_, err = file.Write(pwData)
	}

	return tempFileName, err
}

// NewPasswordFileStore creates a new instance of a file based username/password store.
// Call Open/Release to start/stop using this store.
// Note: this store is intended for one writer and many readers.
// Multiple concurrent writes are not supported and might lead to one write being ignored.
//
//	filepath location of the file store. See also DefaultPasswordFile for the recommended name
func NewPasswordFileStore(filepath string) *PasswordFileStore {
	store := &PasswordFileStore{
		storePath: filepath,
		hashAlgo:  PWHASH_ARGON2id,
		entries:   make(map[string]PasswordEntry),
	}
	return store
}
