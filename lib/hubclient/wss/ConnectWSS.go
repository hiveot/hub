package wssclient

import (
	"context"
	"crypto/x509"
	"fmt"
	"github.com/coder/websocket"
	"github.com/hiveot/hub/lib/tlsclient"
	"log/slog"
	"net/url"
)

// ConnectWSS establishes a websocket session using the given HTTPS client.
func ConnectWSS(
	clientID string, wssURL string, bearerToken string, caCert *x509.Certificate,
	onConnect func(bool, error),
	onMessage func(msg map[string]interface{}),
) (cancelFn func(), conn *websocket.Conn, err error) {

	// separate client with a long timeout for sse
	// use a new http client instance to set an indefinite timeout for the sse connection
	httpClient := tlsclient.NewHttp2TLSClient(caCert, nil, 0)

	slog.Info("ConnectWSS (to hub) - establish Websocket connection to server",
		slog.String("URL", wssURL),
		slog.String("clientID", clientID),
	)

	// use context to disconnect the client
	wssCtx, wssCancelFn := context.WithCancel(context.Background())
	opts := websocket.DialOptions{
		HTTPClient: httpClient,
		// auth header added below
		HTTPHeader: nil,
		// unclear if this should be used
		Subprotocols: nil,
		// future: investigate memory vs performance for many connections and different compression modes
		CompressionMode: websocket.CompressionDisabled,
	}
	opts.HTTPHeader.Add("Authorization", "bearer "+bearerToken)
	parts, _ := url.Parse(wssURL)
	origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	opts.HTTPHeader.Add("Origin", origin)
	wssConn, _, err := websocket.Dial(wssCtx, wssURL, &opts)
	if err != nil {
		wssCancelFn()
		return nil, nil, err
	}

	closeWSSFn := func() {
		wssConn.Close(websocket.StatusNormalClosure, "")
		// is this needed after close above?
		wssCancelFn()
	}
	return closeWSSFn, wssConn, fmt.Errorf("Not implemented")
}
