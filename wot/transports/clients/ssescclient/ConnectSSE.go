package ssescclient

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/wot/transports/clients/httpbinding"
	"github.com/hiveot/hub/wot/transports/utils/tlsclient"
	"github.com/tmaxmax/go-sse"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// maxSSEMessageSize allow this maximum size of an SSE message
// go-sse allows this increase of allocation size on receiving messages
const maxSSEMessageSize = 1024 * 1024 * 10

// ConnectSSE establishes a new sse connection.
//
// If the connection is interrupted, the sse connection retries with backoff period.
// If an authentication error occurs then the onDisconnect handler is invoked with an error.
// If the connection is cancelled then the onDisconnect is invoked without error
func ConnectSSE(
	clientID string, cid string,
	sseURL string, bearerToken string,
	caCert *x509.Certificate,
	onConnect func(bool, error),
	onMessage func(event sse.Event),
) (cancelFn func(), err error) {

	// separate client with a long timeout for sse
	// use a new http client instance to set an indefinite timeout for the sse connection
	httpClient := tlsclient.NewHttp2TLSClient(caCert, nil, 0)

	slog.Info("ConnectSSE (to hub) - establish SSE connection to server",
		slog.String("URL", sseURL),
		slog.String("clientID", clientID),
		slog.String("cid", cid),
	)

	// use context to disconnect the client
	sseCtx, sseCancelFn := context.WithCancel(context.Background())
	bodyReader := bytes.NewReader([]byte{})
	req, err := http.NewRequestWithContext(sseCtx, http.MethodGet, sseURL, bodyReader)
	if err != nil {
		sseCancelFn()
		return nil, err
	}
	req.Header.Add(httpbinding.ConnectionIDHeader, cid)
	req.Header.Add("Authorization", "bearer "+bearerToken)
	parts, _ := url.Parse(sseURL)
	origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	req.Header.Add("Origin", origin)
	//req.Header.Add("Connection", "keep-alive")

	sseClient := &sse.Client{
		HTTPClient: httpClient,
		OnRetry: func(err error, _ time.Duration) {
			slog.Info("SSE Connection retry", "err", err, "clientID", clientID)
			// TODO: how to be notified if the connection is restored?
			//  workaround: in handleSSEEvent, update the connection status
			onConnect(false, err)
		},
	}
	conn := sseClient.NewConnection(req)

	// increase the maximum buffer size to 1M (_maxSSEMessageSize)
	// note this requires go-sse v0.9.0-pre.2 as a minimum.
	//https://github.com/tmaxmax/go-sse/issues/32
	newBuf := make([]byte, 0, 1024*65)
	// TODO: make limit configurable
	conn.Buffer(newBuf, maxSSEMessageSize)
	remover := conn.SubscribeToAll(onMessage)

	go func() {
		// connect and wait until the connection ends
		// and report an error if connection ends due to reason other than context cancelled
		// onConnect will be called on receiving the first (ping) message
		//onConnect(true, nil)
		err := conn.Connect()
		onConnect(false, err)

		if connError, ok := err.(*sse.ConnectionError); ok {
			// since sse retries, this is likely an authentication error
			slog.Error("SSE connection failed (server shutdown or connection interrupted)",
				"clientID", clientID,
				"err", err.Error())
			_ = connError
			err = fmt.Errorf("Reconnect Failed: %w", connError.Err) //connError.Err
		} else if errors.Is(err, context.Canceled) {
			// context was cancelled. no error
			err = nil
		}
		// test if we're still receiving events after context is closed
		_ = remover
		//remover() // remove subscriptions connection
		//req.Close()
		//
	}()

	// wait for the SSE connection to be established
	// If an RPC action is sent too early then no reply will be received.
	time.Sleep(time.Millisecond * 10)

	closeSSEFn := func() {
		// any other cleanup?
		sseCancelFn()
	}

	return closeSSEFn, nil
}
