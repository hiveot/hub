package unpwstore_test

import (
	"fmt"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"golang.org/x/exp/slog"
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

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	tempFolder = path.Join(os.TempDir(), "hiveot-authn-test")
	_ = os.MkdirAll(tempFolder, 0700)

	// Connect without pw file
	unpwFilePath = path.Join(tempFolder, unpwFileName)
	os.Remove(unpwFilePath)

	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(tempFolder)
	}
	os.Exit(res)
}

func TestOpenClosePWFile(t *testing.T) {
	unpwStore := unpwstore.NewPasswordFileStore(unpwFilePath)
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
	unpwStore := unpwstore.NewPasswordFileStore("/bin/yes")
	err := unpwStore.Open()
	assert.Error(t, err)

}

func TestGetMissingEntry(t *testing.T) {
	const user1 = "user1"
	// create 2 separate stores
	pwStore1 := unpwstore.NewPasswordFileStore(unpwFilePath)
	err := pwStore1.Open()
	require.NoError(t, err)
	defer pwStore1.Close()

	// entry doesn't yet exist
	entry, err := pwStore1.GetEntry("iddoesn'texist")
	assert.Error(t, err)
	assert.Empty(t, entry)
}

// verify password
func TestVerify(t *testing.T) {
	const user1 = "user1"
	const pass1 = "pass1"
	os.Remove(unpwFilePath)
	pwStore1 := unpwstore.NewPasswordFileStore(unpwFilePath)
	err := pwStore1.Open()
	require.NoError(t, err)
	defer pwStore1.Close()

	// entry doesn't yet exist
	pwStore1.SetPassword(user1, pass1)
	entry, err := pwStore1.GetEntry(user1)
	assert.NoError(t, err)
	assert.Equal(t, user1, entry.LoginID)
	assert.NotEmpty(t, entry.Updated)
	assert.NotEmpty(t, entry.PasswordHash)

	// verify correct password
	err = pwStore1.VerifyPassword(user1, pass1)
	assert.NoError(t, err)

	// empty password not allowed
	err = pwStore1.SetPassword(user1, "")
	assert.NoError(t, err)
	err = pwStore1.VerifyPassword(user1, "")
	assert.Error(t, err)

	// wrong user not found
	err = pwStore1.VerifyPassword("wronguser", pass1)
	assert.Error(t, err)
	// wrong password should fail
	err = pwStore1.VerifyPassword(user1, "wrongpassword")
	assert.Error(t, err)
}

func TestName(t *testing.T) {
	const user1 = "user1"
	const name1 = "user one"
	os.Remove(unpwFilePath)
	pwStore1 := unpwstore.NewPasswordFileStore(unpwFilePath)
	err := pwStore1.Open()
	require.NoError(t, err)
	defer pwStore1.Close()

	// entry doesn't yet exist
	pwStore1.SetName(user1, name1)
	entry, err := pwStore1.GetEntry(user1)
	assert.NoError(t, err)
	assert.Equal(t, user1, entry.LoginID)
	assert.Equal(t, name1, entry.UserName)
	assert.NotEmpty(t, entry.Updated)
	assert.Empty(t, entry.PasswordHash)
}

func TestSetPasswordTwoStores(t *testing.T) {
	const user1 = "user1"
	const user2 = "user2"
	const hash1 = "hash1"
	const hash2 = "hash2"

	// create 2 separate stores
	pwStore1 := unpwstore.NewPasswordFileStore(unpwFilePath)
	err := pwStore1.Open()
	assert.NoError(t, err)
	pwStore2 := unpwstore.NewPasswordFileStore(unpwFilePath)
	err = pwStore2.Open()
	assert.NoError(t, err)

	// set hash in store 1, should appear in store 2
	err = pwStore1.SetPasswordHash(user1, hash1)
	assert.NoError(t, err)
	// wait for reload
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
	exists := pwStore2.Exists(user1)
	assert.True(t, exists)

	entry, err := pwStore2.GetEntry(user1)
	assert.NoError(t, err)
	assert.Equal(t, hash1, entry.PasswordHash)

	// do it again but in reverse
	slog.Info("- do it again in reverse -")
	err = pwStore2.SetPasswordHash(user2, hash2)
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 100)

	entry, err = pwStore1.GetEntry(user2)
	assert.Equal(t, hash2, entry.PasswordHash)

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
	pwStore1 := unpwstore.NewPasswordFileStore(unpwFilePath)
	err := pwStore1.Open()
	assert.NoError(t, err)
	pwStore2 := unpwstore.NewPasswordFileStore(unpwFilePath)
	err = pwStore2.Open()
	assert.NoError(t, err)

	wg.Add(1)
	go func() {
		for i = 0; i < 30; i++ {
			thingID := fmt.Sprintf("thing-%d", i)
			err2 := pwStore1.SetPasswordHash(thingID, "hash1")
			time.Sleep(time.Millisecond * 1)
			if err2 != nil {
				assert.NoError(t, err2)
			}
		}
		wg.Done()
	}()
	wg.Wait()
	// time to catch up the file watcher debouncing
	time.Sleep(time.Second * 3)

	// both stores should be fully up to date
	assert.Equal(t, i, pwStore1.Count())
	assert.Equal(t, i, pwStore2.Count())

	//
	pwStore1.Close()
	pwStore2.Close()
}

func TestWritePwToBadTempFolder(t *testing.T) {
	pws := make(map[string]unpwstore.PasswordEntry)
	pwStore1 := unpwstore.NewPasswordFileStore(unpwFilePath)
	err := pwStore1.Open()
	assert.NoError(t, err)
	_, err = unpwstore.WritePasswordsToTempFile("/badfolder", pws)
	assert.Error(t, err)
	pwStore1.Close()
}

func TestWritePwToReadonlyFile(t *testing.T) {
	const user1 = "user1"
	const pass1 = "pass1"
	// bin/yes cannot be written to
	pwStore1 := unpwstore.NewPasswordFileStore("/bin/yes")
	err := pwStore1.Open()
	assert.Error(t, err)
	err = pwStore1.SetPassword(user1, pass1)
	assert.Error(t, err)
	pwStore1.Close()
}
