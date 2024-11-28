package wssbinding

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/tlsclient"
	"log/slog"
	"net/url"
)

// Paths used by this protocol binding - SYNC with HttpBindingClient.ts
//
// THIS WILL BE REMOVED AFTER THE PROTOCOL BINDING PUBLISHES THESE IN THE TDD.
// The hub client will need the TD (ConsumedThing) to determine the paths.
const (

	// deprecated authn service - use the generated constants or forms
	PostLoginPath = "/authn/login"
	// deprecated authn service - use the generated constants
	PostLogoutPath = "/authn/logout"
	// deprecated authn service - use the generated constants
	PostRefreshPath = "/authn/refresh"
)

// ConnectWithPassword connects to the Hub TLS server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
func (cl *WssTransportClient) ConnectWithPassword(password string) (newToken string, err error) {
	//cl.mux.Lock()
	//// remove existing connection
	//if cl.tlsClient != nil {
	//	cl.tlsClient.Close()
	//}
	//cl.tlsClient = tlsclient.NewTLSClient(
	//	cl.hostPort, nil, cl.caCert, cl.timeout, cl.cid)
	wssURI, err := url.Parse(cl.wssURL)
	loginURL := fmt.Sprintf("https://%s%s",
		wssURI.Host,
		PostLoginPath)
	//cl.mux.Unlock()

	slog.Info("ConnectWithPassword", "clientID", cl.clientID)

	// FIXME: figure out how a standard login method is used to obtain an auth token
	loginMessage := map[string]string{
		"login":    cl.GetClientID(),
		"password": password,
	}
	// TODO: this is part of the http binding, not the websocket binding
	// a sacrificial client to get a token
	tlsClient := tlsclient.NewTLSClient(wssURI.Host, nil, cl.caCert, cl.timeout, "")
	argsJSON, _ := json.Marshal(loginMessage)
	resp, _, statusCode, _, err2 := tlsClient.Invoke(
		"POST", loginURL, argsJSON, "", nil)
	if err2 != nil {
		err = fmt.Errorf("%d: Login failed: %s", statusCode, err2)
		return "", err
	}
	token := ""
	err = cl.Unmarshal(resp, &token)
	if err != nil {
		err = fmt.Errorf("ConnectWithPassword: Login to %s has unexpected response message: %s", loginURL, err)
		return "", err
	}
	// with an auth token the connection can be established

	cl.wssCancelFn, cl.wssConn, err = ConnectWSS(
		cl.clientID, cl.wssURL, token, cl.caCert,
		cl._onConnect, cl.handleWSSMessage)

	if err == nil {
		cl.token = token
		cl.retryOnDisconnect.Store(true)
	}

	return token, err
}

// ConnectWithToken connects to the Hub server using a user bearer token
// and obtain a new token.
//
//	token is the token previously obtained with login or refresh.
func (cl *WssTransportClient) ConnectWithToken(token string) (newToken string, err error) {

	cl.wssCancelFn, cl.wssConn, err = ConnectWSS(
		cl.clientID, cl.wssURL, token, cl.caCert,
		cl._onConnect, cl.handleWSSMessage)

	if err == nil {
		// Refresh the auth token and verify the connection works.
		newToken, err = cl.RefreshToken(token)
	}
	// once the connection is established enable the retry on disconnect
	if err == nil {
		cl.retryOnDisconnect.Store(true)
	}
	return newToken, err
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithToken'
// This is specific to the Hiveot Hub.
//func (cl *WssTransportClient) RefreshToken(oldToken string) (newToken string, err error) {
//
//	// FIXME: what is the standard for refreshing a token using http?
//	slog.Info("RefreshToken", slog.String("clientID", cl.clientID))
//	refreshURL := fmt.Sprintf("https://%s%s", cl.hostPort, PostRefreshPath)
//
//	// the bearer token holds the old token
//	resp, _, err := cl._send(
//		"POST", refreshURL, "", nil, nil)
//
//	// set the new token as the bearer token
//	if err == nil {
//		err = jsoniter.Unmarshal(resp, &newToken)
//
//		if err == nil {
//			// reconnect using the new token
//			cl.mux.Lock()
//			cl.bearerToken = newToken
//			cl.mux.Unlock()
//		}
//	}
//	return newToken, err
//}

// Logout from the server and end the session.
// This is specific to the Hiveot Hub.
func (cl *WssTransportClient) Logout() error {

	// TODO: find a way to derive this from a form
	slog.Info("Logout", slog.String("clientID", cl.clientID))

	// TODO: this is part of the http binding, not the websocket binding
	// a sacrificial client to logout
	wssURI, _ := url.Parse(cl.wssURL)
	tlsClient := tlsclient.NewTLSClient(wssURI.Host, nil, cl.caCert, cl.timeout, "")
	tlsClient.SetAuthToken(cl.token)
	_, _, _, _, err2 := tlsClient.Invoke(
		"POST", PostLogoutPath, nil, "", nil)
	if err2 != nil {
		err := fmt.Errorf("logout failed: %s", err2)
		return err
	}
	return nil
}
