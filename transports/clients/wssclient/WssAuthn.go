package wssclient

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils/tlsclient"
	"log/slog"
)

// ConnectWithPassword connects to the Hub TLS server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
func (cl *WssTransportClient) ConnectWithPassword(password string) (newToken string, err error) {
	// Login using the http endpoint
	loginURL := fmt.Sprintf("https://%s%s", cl.BaseHostPort, transports.HttpPostLoginPath)

	slog.Info("ConnectWithPassword", "clientID", cl.BaseClientID)

	// FIXME: figure out how a standard login method is used to obtain an auth token
	loginMessage := map[string]string{
		"login":    cl.GetClientID(),
		"password": password,
	}
	// TODO: this is part of the http binding, not the websocket binding
	// a sacrificial client to get a token
	tlsClient := tlsclient.NewTLSClient(
		loginURL, nil, cl.BaseCaCert, cl.BaseTimeout, "")
	argsJSON, _ := json.Marshal(loginMessage)
	defer tlsClient.Close()
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
		cl.BaseClientID, cl.BaseFullURL, token, cl.BaseCaCert,
		cl._onConnect, cl.WssClientHandleMessage)

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

	wssCancelFn, wssConn, err := ConnectWSS(
		cl.BaseClientID, cl.BaseFullURL, token, cl.BaseCaCert,
		cl._onConnect, cl.WssClientHandleMessage)

	cl.BaseMux.Lock()
	cl.wssCancelFn = wssCancelFn
	cl.wssConn = wssConn
	cl.BaseMux.Unlock()

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
// This uses the http method
// The resulting token can be used with 'ConnectWithToken'
// This is specific to the Hiveot Hub.
func (cl *WssTransportClient) RefreshToken(oldToken string) (newToken string, err error) {
	// use the http endpoint to refresh the token
	refreshURL := fmt.Sprintf("https://%s%s", cl.BaseHostPort, transports.HttpPostRefreshPath)

	// TODO: this is part of the http binding, not the websocket binding
	// a sacrificial client to get a token
	tlsClient := tlsclient.NewTLSClient(
		cl.BaseHostPort, nil, cl.BaseCaCert, cl.BaseTimeout, "")
	tlsClient.SetAuthToken(oldToken)
	defer tlsClient.Close()

	argsJSON, _ := json.Marshal(oldToken)
	resp, _, statusCode, _, err2 := tlsClient.Invoke(
		"POST", refreshURL, argsJSON, "", nil)
	if err2 != nil {
		err = fmt.Errorf("%d: Refresh failed: %s", statusCode, err2)
		return "", err
	}
	err = cl.Unmarshal(resp, &newToken)
	if err != nil {
		return "", err
	}
	cl.token = newToken
	return newToken, err
}

// Logout from the server and end the session.
// This is specific to the Hiveot Hub.
func (cl *WssTransportClient) Logout() error {

	// TODO: find a way to derive this from a form
	slog.Info("Logout", slog.String("clientID", cl.BaseClientID))

	// Use a sacrificial http client to logout
	tlsClient := tlsclient.NewTLSClient(
		cl.BaseHostPort, nil, cl.BaseCaCert, cl.BaseTimeout, "")
	tlsClient.SetAuthToken(cl.token)
	defer tlsClient.Close()

	logoutURL := fmt.Sprintf("https://%s%s", cl.BaseHostPort, transports.HttpPostLogoutPath)
	_, _, _, _, err2 := tlsClient.Invoke(
		"POST", logoutURL, nil, "", nil)
	if err2 != nil {
		err := fmt.Errorf("logout failed: %s", err2)
		return err
	}
	return nil
}
