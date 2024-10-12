package authnstore_test

import (
	"fmt"
	authn2 "github.com/hiveot/hub/api/go/authn"
	authz2 "github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"github.com/hiveot/hub/runtime/authn/config"
	"log/slog"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

const unpwFileName = "testunpwstore.passwd"

var unpwFilePath string

var tempFolder string
var algo = config.PWHASH_ARGON2id

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	tempFolder = path.Join(os.TempDir(), "hiveot-authn-test")
	_ = os.MkdirAll(tempFolder, 0700)

	// Connect without pw file
	unpwFilePath = path.Join(tempFolder, unpwFileName)
	_ = os.Remove(unpwFilePath)

	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(tempFolder)
	}
	os.Exit(res)
}

func TestOpenClosePWFile(t *testing.T) {
	_ = os.Remove(unpwFilePath)
	unpwStore := authnstore.NewAuthnFileStore(unpwFilePath, "")
	err := unpwStore.Open()
	assert.NoError(t, err)

	// open twice should provide error
	err2 := unpwStore.Open()
	assert.Error(t, err2)

	time.Sleep(time.Millisecond * 100)
	unpwStore.Close()
}

func TestOpenBadData(t *testing.T) {
	// /bin/yes cannot be read
	unpwStore := authnstore.NewAuthnFileStore("/bin/yes", "")
	err := unpwStore.Open()
	assert.Error(t, err)

}

func TestGetMissingEntry(t *testing.T) {
	const user1 = "user1"
	_ = os.Remove(unpwFilePath)
	// create 2 separate stores
	pwStore1 := authnstore.NewAuthnFileStore(unpwFilePath, "")
	err := pwStore1.Open()
	require.NoError(t, err)
	defer pwStore1.Close()

	// entry doesn't yet exist
	entry, err := pwStore1.GetProfile("iddoesn'texist")
	assert.Error(t, err)
	assert.Empty(t, entry)
}

// Add password
func TestAdd(t *testing.T) {
	const user1 = "user1"
	const pass1 = "pass1"
	const user2 = "user2"
	const pass2 = "pass2"
	_ = os.Remove(unpwFilePath)
	pwStore1 := authnstore.NewAuthnFileStore(unpwFilePath, "")
	err := pwStore1.Open()
	require.NoError(t, err)
	defer pwStore1.Close()

	// add should succeed
	err = pwStore1.Add(user1, authn2.ClientProfile{
		ClientID:   user1,
		ClientType: authn2.ClientTypeConsumer,
	})
	require.NoError(t, err)

	// adding existing user should update it
	err = pwStore1.Add(user1, authn2.ClientProfile{
		ClientID:    user1,
		ClientType:  authn2.ClientTypeConsumer,
		DisplayName: "updated user 1",
	})
	assert.NoError(t, err)
	prof1, _ := pwStore1.GetProfile(user1)
	assert.Equal(t, "updated user 1", prof1.DisplayName)

	// adding missing client should fail
	err = pwStore1.Add(user2, authn2.ClientProfile{ClientID: "", ClientType: authn2.ClientTypeConsumer})
	assert.Error(t, err)
	// adding missing type should fail
	err = pwStore1.Add(user2, authn2.ClientProfile{ClientID: user2, ClientType: ""})
	assert.Error(t, err)

}

// verify password
func TestVerifyHashAlgo(t *testing.T) {
	const user1 = "user1"
	const pass1 = "pass1"
	const user2 = "user2"
	const pass2 = "pass2"

	_ = os.Remove(unpwFilePath)
	pwStore1 := authnstore.NewAuthnFileStore(unpwFilePath, algo)
	err := pwStore1.Open()
	require.NoError(t, err)
	defer pwStore1.Close()

	err = pwStore1.Add(user1, authn2.ClientProfile{
		ClientID: user1, ClientType: authn2.ClientTypeConsumer})
	err = pwStore1.Add(user2, authn2.ClientProfile{
		ClientID: user2, ClientType: authn2.ClientTypeConsumer})
	err = pwStore1.SetPassword(user1, pass1)
	require.NoError(t, err)
	profile1, err := pwStore1.GetProfile(user1)
	require.NoError(t, err)
	profile2, err2 := pwStore1.VerifyPassword(user1, pass1)
	require.NoError(t, err2, "password verification failed")
	assert.Equal(t, user1, profile1.ClientID)
	assert.Equal(t, profile1, profile2)
	assert.NotEmpty(t, profile1.Updated)

	// verify incorrect password
	_, err = pwStore1.VerifyPassword(user1, pass2)
	assert.Error(t, err)

	// empty password not allowed
	err = pwStore1.SetPassword(user1, "")
	assert.Error(t, err)
	// empty password verification should fail
	_, err = pwStore1.VerifyPassword(user2, "")
	assert.Error(t, err)

	// wrong user not found
	_, err = pwStore1.VerifyPassword("wronguser", pass1)
	assert.Error(t, err)
	// wrong password should fail
	_, err = pwStore1.VerifyPassword(user1, "wrongpassword")
	assert.Error(t, err)

	// after removing user it should fail

	// verify incorrect password
	err = pwStore1.Remove(user1)
	assert.NoError(t, err)
	_, err = pwStore1.VerifyPassword(user1, pass1)
	assert.Error(t, err)
}

// verify password
func TestVerifyBCryptAlgo(t *testing.T) {
	algo = config.PWHASH_BCRYPT
	TestVerifyHashAlgo(t)
	algo = config.PWHASH_ARGON2id
}

func TestName(t *testing.T) {
	const user1 = "user1"
	const name1 = "user one"

	_ = os.Remove(unpwFilePath)
	pwStore1 := authnstore.NewAuthnFileStore(unpwFilePath, "")
	err := pwStore1.Open()
	require.NoError(t, err)
	defer pwStore1.Close()
	err = pwStore1.Add(user1, authn2.ClientProfile{
		ClientID: user1, ClientType: authn2.ClientTypeConsumer, DisplayName: name1})

	entry, err := pwStore1.GetProfile(user1)
	assert.NoError(t, err)
	assert.Equal(t, user1, entry.ClientID)
	assert.Equal(t, name1, entry.DisplayName)
	assert.NotEmpty(t, entry.Updated)
}

func TestSetPasswordTwoStores(t *testing.T) {
	const user1 = "user1"
	const user2 = "user2"
	const pass1 = "pass1"
	const pass2 = "pass2"

	// create 2 separate stores
	_ = os.Remove(unpwFilePath)
	pwStore1 := authnstore.NewAuthnFileStore(unpwFilePath, "")
	err := pwStore1.Open()
	require.NoError(t, err)
	err = pwStore1.Add(user1,
		authn2.ClientProfile{ClientID: user1, ClientType: authn2.ClientTypeConsumer})
	require.NoError(t, err)
	//
	pwStore2 := authnstore.NewAuthnFileStore(unpwFilePath, "")
	err = pwStore2.Open()
	require.NoError(t, err)
	err = pwStore2.Add(user2,
		authn2.ClientProfile{ClientID: user2, ClientType: authn2.ClientTypeConsumer})
	require.NoError(t, err)
	err = pwStore1.Reload()
	require.NoError(t, err)
	time.Sleep(time.Millisecond)

	// set password in store 1, should appear in store 2
	err = pwStore1.SetPassword(user1, pass1)
	assert.NoError(t, err)
	// wait before reload
	time.Sleep(time.Millisecond * 100)
	// check mode of pw file
	info, err := os.Stat(unpwFilePath)
	require.NoError(t, err)
	mode := info.Mode()
	assert.Equal(t, 0600, int(mode), "file mode not 0600")

	// read back
	// force reload. Don't want to wait
	err = pwStore2.Reload()
	assert.NoError(t, err)

	// must exist
	_, err = pwStore2.GetProfile(user1)
	assert.NoError(t, err)

	profile1, err := pwStore2.GetProfile(user1)
	assert.NoError(t, err)
	profile2, err := pwStore2.VerifyPassword(user1, pass1)
	assert.NoError(t, err)
	assert.NotEmpty(t, profile1)
	assert.Equal(t, profile1, profile2)

	// do it again but in reverse
	slog.Info("- do it again in reverse -")
	err = pwStore2.SetPassword(user2, pass2)
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 100)

	prof2, err := pwStore1.GetProfile(user2)
	assert.NoError(t, err)
	assert.Equal(t, prof2.ClientID, user2)
	prof3, err := pwStore1.VerifyPassword(user2, pass2)
	assert.Equal(t, prof2, prof3)

	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 100)
	pwStore1.Close()
	pwStore2.Close()
}

// Load test if one writer with second reader
func TestConcurrentReadWrite(t *testing.T) {
	var wg sync.WaitGroup
	var i int

	// start with empty file
	fp, _ := os.Create(unpwFilePath)
	_ = fp.Close()

	// two stores in parallel
	pwStore1 := authnstore.NewAuthnFileStore(unpwFilePath, "")
	err := pwStore1.Open()
	assert.NoError(t, err)
	pwStore2 := authnstore.NewAuthnFileStore(unpwFilePath, "")
	err = pwStore2.Open()
	assert.NoError(t, err)

	wg.Add(1)
	go func() {
		for i = 0; i < 30; i++ {
			thingID := fmt.Sprintf("things-%d", i)
			err = pwStore1.Add(thingID,
				authn2.ClientProfile{ClientID: thingID, ClientType: authn2.ClientTypeConsumer})
			time.Sleep(time.Millisecond * 1)
			if err != nil {
				assert.NoError(t, err)
			}
		}
		wg.Done()
	}()
	wg.Wait()
	// time to catch up the file watcher debouncing
	time.Sleep(time.Second * 1)

	profiles, err := pwStore1.GetProfiles()
	assert.NoError(t, err)

	// both stores should be fully up to date
	assert.Equal(t, i, len(profiles))
	assert.Equal(t, i, pwStore1.Count())
	assert.Equal(t, i, pwStore2.Count())

	//
	pwStore1.Close()
	pwStore2.Close()
}

func TestWritePwToBadTempFolder(t *testing.T) {
	pws := make(map[string]api.AuthnEntry)
	pwStore1 := authnstore.NewAuthnFileStore(unpwFilePath, "")
	err := pwStore1.Open()
	assert.NoError(t, err)
	_, err = authnstore.WritePasswordsToTempFile("/badfolder", pws)
	assert.Error(t, err)
	pwStore1.Close()
}

func TestWritePwToReadonlyFile(t *testing.T) {
	const user1 = "user1"
	const pass1 = "pass1"
	// bin/yes cannot be written to
	pwStore1 := authnstore.NewAuthnFileStore("/bin/yes", "")
	err := pwStore1.Open()
	assert.Error(t, err)
	err = pwStore1.SetPassword(user1, pass1)
	assert.Error(t, err)
	pwStore1.Close()
}

func TestUpdate(t *testing.T) {
	const user1 = "user1"
	const name1 = "name1"
	const key1 = "pubkey1"
	const user2 = "user2"
	const name2 = "name2"

	_ = os.Remove(unpwFilePath)
	pwStore1 := authnstore.NewAuthnFileStore(unpwFilePath, "")
	err := pwStore1.Open()
	require.NoError(t, err)
	err = pwStore1.Add(user1, authn2.ClientProfile{ClientID: user1,
		ClientType: authn2.ClientTypeConsumer, DisplayName: name1})
	require.NoError(t, err)

	// update must be of the same user
	err = pwStore1.UpdateProfile(user1, authn2.ClientProfile{ClientID: user2, ClientType: authn2.ClientTypeConsumer})
	assert.Error(t, err)

	// update must succeed
	err = pwStore1.UpdateProfile(user1, authn2.ClientProfile{
		ClientID: user1, ClientType: authn2.ClientTypeConsumer,
		DisplayName: name2, PubKey: key1,
	})
	assert.NoError(t, err)
	prof, err := pwStore1.GetProfile(user1)
	assert.NoError(t, err)
	assert.Equal(t, name2, prof.DisplayName)
	assert.Equal(t, key1, prof.PubKey)

	// update of non-existing user should fail
	err = pwStore1.UpdateProfile(
		user1, authn2.ClientProfile{ClientID: "notauser", ClientType: authn2.ClientTypeConsumer})
	assert.Error(t, err)
	err = pwStore1.UpdateProfile(
		"notauser", authn2.ClientProfile{ClientID: user1, ClientType: authn2.ClientTypeConsumer})
	assert.Error(t, err)
}

func TestSetRole(t *testing.T) {
	const user1 = "user1"
	const role1 string = string(authz2.ClientRoleAgent)

	_ = os.Remove(unpwFilePath)
	pwStore1 := authnstore.NewAuthnFileStore(unpwFilePath, "")
	err := pwStore1.Open()
	require.NoError(t, err)
	err = pwStore1.Add(user1, authn2.ClientProfile{ClientID: user1,
		ClientType: authn2.ClientTypeConsumer, DisplayName: "test user"})
	require.NoError(t, err)

	err = pwStore1.SetRole(user1, role1)
	assert.NoError(t, err)
	role2, err := pwStore1.GetRole(user1)
	assert.NoError(t, err)
	assert.Equal(t, role1, role2)
}
