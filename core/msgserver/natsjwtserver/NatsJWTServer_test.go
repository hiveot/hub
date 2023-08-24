package natsjwtserver

import (
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/core/msgserver/natsnkeyserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/svcconfig"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"os"
	"path"
	"testing"
	"time"
)

func TestStartStopJWT(t *testing.T) {
	homeDir := path.Join(os.TempDir(), "nats-server-test")
	f := svcconfig.GetFolders(homeDir, false)

	certBundle := certs.CreateTestCertBundle()
	s := NewNatsJWTServer()
	cfg := natsnkeyserver.NatsServerConfig{
		Port:       9990,
		ServerCert: certBundle.ServerCert,
		CaCert:     certBundle.CaCert,
		CaKey:      certBundle.CaKey,
	}
	err := cfg.Setup(f.Certs, f.Stores, false)
	require.NoError(t, err)
	clientURL, err := s.Start(cfg)
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	c, err := s.ConnectInProc("testjwtservice")
	require.NoError(t, err)
	require.NotEmpty(t, c)
	c.Close()
}

// this requires the JWT server. It cannot be used together with NKeys :/
func TestLoginWithJWT(t *testing.T) {
	slog.Info("--- TestLoginWithJWT start")
	defer slog.Info("--- TestLoginWithJWT end")

	rxMsg := ""
	_, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()

	// raw generate a jwt token
	//userKey, _ := nkeys.CreateUser()
	userKey := serverCfg.CoreServiceKP
	userJWT := serverCfg.CoreServiceJWT
	hc1, err := natshubclient.ConnectWithJWT(clientURL, userKey, userJWT, certBundle.CaCert)
	require.NoError(t, err)

	_, err = hc1.Subscribe("things.>", func(msg *nats.Msg) {
		rxMsg = string(msg.Data)
		slog.Info("received message", "msg", rxMsg)
	})
	assert.NoError(t, err, "unable to subscribe")
	err = hc1.Pub("things.service1.event", []byte("hello world"))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.Equal(t, "hello world", rxMsg)

	hc1.Disconnect()
}

func TestLoginWithInvalidJWT(t *testing.T) {
	slog.Info("--- TestLoginWithInvalidJWT start")
	defer slog.Info("--- TestLoginWithInvalidJWT end")
	_, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()

	// token signed by fake account should fail
	fakeAccountKey, _ := nkeys.CreateAccount()
	userKey, _ := nkeys.CreateUser()
	userPub, _ := userKey.PublicKey()
	userClaims := jwt.NewUserClaims(userPub)
	userClaims.IssuerAccount, _ = fakeAccountKey.PublicKey()
	badToken, _ := userClaims.Encode(fakeAccountKey)
	hc1, err := natshubclient.ConnectWithJWT(clientURL, userKey, badToken, certBundle.CaCert)
	require.Error(t, err)
	require.Empty(t, hc1)

}
