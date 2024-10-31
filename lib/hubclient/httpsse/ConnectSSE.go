package httpsse

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/tmaxmax/go-sse"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// ConnectSSE establishes a sse session over the Hub HTTPS connection.
// All hub messages are send as type ThingMessage, containing thingID, name, payload and sender
//
// If the connection is interrupted, the sse connection retries with backoff period.
// If an authentication error occurs then the onDisconnect handler is invoked with an error.
// If the connection is cancelled then the onDisconnect is invoked without error
func (cl *HttpSSEClient) ConnectSSE(
	sseURL string, bearerToken string, httpClient *http.Client, onConnect func(bool, error)) error {

	slog.Info("ConnectSSE - establish SSE connection with server",
		slog.String("sseURL", sseURL),
		slog.String("clientID", cl.clientID),
		slog.String("cid", cl.cid))

	// use context to disconnect the client
	sseCtx, sseCancelFn := context.WithCancel(context.Background())
	bodyReader := bytes.NewReader([]byte{})
	req, err := http.NewRequestWithContext(sseCtx, http.MethodGet, sseURL, bodyReader)
	if err != nil {
		sseCancelFn()
		return err
	}
	req.Header.Add(hubclient.ConnectionIDHeader, cl.cid)
	req.Header.Add("Authorization", "bearer "+bearerToken)
	parts, _ := url.Parse(sseURL)
	origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	req.Header.Add("Origin", origin)
	//req.Header.Add("Connection", "keep-alive")

	cl.sseCancelFn = sseCancelFn
	sseClient := &sse.Client{
		HTTPClient: httpClient,
		OnRetry: func(err error, _ time.Duration) {
			slog.Info("SSE Connection retry", "err", err, "clientID", cl.clientID)
			// TODO: how to be notified if the connection is restored?
			//  workaround: in handleSSEEvent, update the connection status
			cl.handleSSEConnect(false, err)
		},
	}
	conn := sseClient.NewConnection(req)

	// increase the maximum buffer size to 1M (_maxSSEMessageSize)
	// note this requires go-sse v0.9.0-pre.2 as a minimum.
	//https://github.com/tmaxmax/go-sse/issues/32
	newBuf := make([]byte, 0, 1024*65)
	// TODO: make limit configurable
	conn.Buffer(newBuf, cl.maxSSEMessageSize)

	remover := conn.SubscribeToAll(cl.handleSSEEvent)
	go func() {
		// connect and wait until the connection ends
		// and report an error if connection ends due to reason other than context cancelled
		onConnect(true, nil)
		err := conn.Connect()
		onConnect(false, err)

		if connError, ok := err.(*sse.ConnectionError); ok {
			// since sse retries, this is likely an authentication error
			slog.Error("SSE connection failed (server shutdown or connection interrupted)",
				"clientID", cl.clientID,
				"err", err.Error())
			_ = connError
			err = fmt.Errorf("Reconnect Failed: %w", connError.Err) //connError.Err
		} else if errors.Is(err, context.Canceled) {
			// context was cancelled. no error
			err = nil
		}
		remover() // cleanup connection
		//
	}()
	// FIXME: wait for the SSE connection to be established
	// If an RPC action is sent too early then no reply will be received.
	time.Sleep(time.Millisecond * 10)
	return nil
}
