package session

import (
	"fmt"
	"log/slog"
	"net/http"
)

// SseHandler handles incoming SSE connections, eg. one per browser tab.
// This subscribes to the user's session sse channel and sends messages to the browser.
// Sse requests are refused if no auth info is found.
func SseHandler(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE response
	//w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
	w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate, max-age=0, no-transform")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")

	// An active session is required before accepting the request
	cs, claims, err := sessionmanager.GetSessionFromCookie(r)
	_ = claims
	if cs == nil || !cs.IsActive() || err != nil {
		slog.Warn("No session available, delay retry to 30 seconds")

		// set retry to a large number
		// while this doesn't redirect, it does stop it from holding a connection.
		// see https://javascript.info/server-sent-events#reconnection
		_, _ = fmt.Fprintf(w, "retry: %s\nevent:%s\n\n",
			"30000", "logout")
		// this result code doesn't seem to work?
		w.WriteHeader(http.StatusUnauthorized)
		w.(http.Flusher).Flush()
		return
	}

	// establish a client event channel
	sseChan := make(chan SSEEvent)
	cs.AddSSEClient(sseChan)

	slog.Info("SseHandler. New SSE connection",
		slog.String("RemoteAddr", r.RemoteAddr),
		slog.String("clientID", cs.clientID),
		slog.Int("nr sse connections", len(cs.sseClients)),
	)
	//var sseMsg SSEEvent

	done := false
	for !done { // sseMsg := range sseChan {
		// wait for message, or writer closing
		select {
		case sseMsg, ok := <-sseChan: // received event
			slog.Debug("SseHandler: received event from sseChan",
				slog.String("remote", r.RemoteAddr),
				slog.String("clientID", cs.clientID),
				slog.String("event", sseMsg.Event),
				slog.Bool("ok", ok),
			)
			if !ok { // channel was closed by session
				done = true
				break
			}
			// WARNING: messages are send as MIME type "text/event-stream", which is defined as
			// "Each message is sent as a block of text terminated by a pair of newlines. "
			//https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events
			//_, err := fmt.Fprintf(w, "event: time\ndata: <div sse-swap='time'>%s</div>\n\n", data)
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
				sseMsg.Event, sseMsg.Payload)
			//_, err := fmt.Fprint(w, sseMsg)
			w.(http.Flusher).Flush()
			break
		case <-r.Context().Done(): // remote client connection closed
			slog.Info("Remote client disconnected (read context)")
			close(sseChan)
			done = true
			break
		}
	}
	cs.RemoveSSEClient(sseChan)

	slog.Info("SseHandler: sse connection closed",
		slog.String("remote", r.RemoteAddr),
		slog.String("clientID", cs.clientID),
		slog.Int("nr sse connections", len(cs.sseClients)),
	)
}
