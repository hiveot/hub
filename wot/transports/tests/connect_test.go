package tests

import (
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/clients"
	"github.com/stretchr/testify/require"
	"testing"
)

const testClient1ID = "client1"
const testClient1Password = "password1"
const serverURL = "https://localhost:12345"

var certBundle = certs.CreateTestCertBundle()

func NewBinding(protocol string) (transports.IProtocolBindingClient, error) {
	clientID := testClient1ID
	fullURL := serverURL
	caCert := certBundle.CaCert
	bc, err := clients.CreateBindingClient(protocol, fullURL, clientID, caCert)
	return bc, err
}

// test connecting to a protocol server
func TestConnect(t *testing.T) {
	binding, err := NewBinding(transports.ProtocolTypeHTTPS)

	token, err := binding.ConnectWithPassword(testClient1Password)
	require.NoError(t, err)
	require.NotEmpty(t, token)

}
