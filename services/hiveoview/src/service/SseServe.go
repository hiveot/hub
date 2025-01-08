package service

import (
	"fmt"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/transports/servers/ssescserver"
	"log/slog"
	"net/http"
)

// SseServe serves incoming SSE connections, eg. one per browser tab.
// This subscribes to the user's session sse channel and sends messages to the browser.
// Sse requests are refused if no auth info is found.
//
// When the connection is established this sends a ping message so the client
// can confirm it is connected.
func SseServe(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE response
	//w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
	w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate, max-age=0, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")

	// An active session is required before accepting the request
	_, cs, err := session.GetSessionFromContext(r)
	if cs == nil || err != nil {
		slog.Warn("SSE Connection attempt but no session exists. Delay retry to 10 seconds",
			"remoteAddr", r.RemoteAddr)

		// set retry to a large number
		// while this doesn't redirect, it does stop it from holding a connection.
		// see https://javascript.info/server-sent-events#reconnection
		_, _ = fmt.Fprintf(w, "retry: %s\nevent:%s\n\n",
			"10000", "logout")
		// this result code doesn't seem to work?
		w.WriteHeader(http.StatusUnauthorized)
		w.(http.Flusher).Flush()
		return
	}

	// request the session event channel
	sseChan := cs.NewSseChan()
	// _send a ping event as the go-sse client doesn't have a 'connected callback'
	// (borrow the event name from the transports SSE server)
	pingEvent := session.SSEEvent{Event: ssescserver.SSEPingEvent}
	sseChan <- pingEvent

	clientID := cs.GetHubClient().GetClientID()

	slog.Debug("SseServe. New SSE incoming connection",
		slog.String("clientID", clientID),
		slog.String("clcid", cs.GetCLCID()),
		slog.String("RemoteAddr", r.RemoteAddr),
	)
	//var sseMsg SSEEvent

	done := false

	// close the channel when the connection drops
	go func() {
		select {
		case <-r.Context().Done(): // remote client connection closed
			slog.Debug("Remote client disconnected (read context)")
			// close channel in the background when no-one is writing
			// in the meantime keep reading. (DeleteSSEChan uses mutex lock)
			go cs.HandleWebConnectionClosed()
		}
	}()

	// read the message channel until it closes
	for !done { // sseMsg := range sseChan {
		// wait for message, or writer closing
		select {
		case sseMsg, ok := <-sseChan: // received event
			slog.Debug("SseServe: received event from sseChan",
				slog.String("remote", r.RemoteAddr),
				slog.String("clientID", clientID),
				slog.String("event", sseMsg.Event),
				slog.Bool("ok", ok),
			)
			if !ok { // channel was closed by session
				// ending the read loop and returning will close the connection
				done = true
				break
			}
			// WARNING: messages are send as MIME type "text/event-stream", which is defined as
			// "Each message is sent as a block of text terminated by a pair of newlines. "
			//https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events
			//_, err := fmt.Fprintf(w, "event: time\ndata: <div sse-swap='time'>%s</div>\n\n", data)
			_, _ = fmt.Fprintf(w, "event: %s\nid: %s\ndata: %s\n\n",
				sseMsg.Event, sseMsg.ID, sseMsg.Payload)
			// ignore write errors as the channel might be closing and must
			// be read to avoid deadlock.
			w.(http.Flusher).Flush()
		}
	}

	//slog.Info("SseServe: sse connection closed",
	//	slog.String("remote", r.RemoteAddr),
	//	slog.String("clientID", clientID),
	//)
}
