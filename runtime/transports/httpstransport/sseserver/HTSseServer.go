package sseserver

import (
	"fmt"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sessions"
	"log/slog"
	"net/http"
)

// HTServeHttp handleSseConnect handles incoming SSE connections, authenticates the client
// Sse requests are refused if no valid session found.
func HTServeHttp(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE response
	//w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate, max-age=0, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")

	// An active session is required before accepting the request. This is created on
	// authentication/login.
	cs, err := sessions.GetSessionFromContext(r)
	if cs == nil || err != nil {
		slog.Warn("No session available, telling client to delay retry to 30 seconds")

		// set retry to a large number
		// while this doesn't redirect, it does stop it from holding a connection.
		// see https://javascript.info/server-sent-events#reconnection
		//_, _ = fmt.Fprintf(w, "retry: %s\nevent:%s\n\n",
		//	"30000", "logout")
		//w.WriteHeader(http.StatusUnauthorized)

		// NOTE: the above works while the below does not???
		errMsg := fmt.Sprintf("retry: %s\nevent:%s\n\n",
			"30000", "logout")
		http.Error(w, errMsg, http.StatusUnauthorized)
		//w.Write([]byte(errMsg))
		w.(http.Flusher).Flush()
		return
	}

	// establish a client event channel for sending messages back to the client
	sseChan := cs.CreateSSEChan()

	// TODO: if this is a first connection of the client send a connected event

	slog.Info("SseHandler. New SSE connection",
		slog.String("RemoteAddr", r.RemoteAddr),
		slog.String("clientID", cs.GetClientID()),
		slog.String("protocol", r.Proto),
		slog.Int("nr sse connections", cs.GetNrConnections()),
	)
	//var sseMsg SSEEvent

	done := false
	for !done { // sseMsg := range sseChan {
		// wait for message, or writer closing
		select {
		case sseMsg, ok := <-sseChan: // received event

			if !ok { // channel was closed by session
				done = true
				break
			}
			slog.Info("SseHandler: sending sse event to client",
				slog.String("remote", r.RemoteAddr),
				slog.String("clientID", cs.GetClientID()),
				slog.String("eventType", sseMsg.EventType),
			)
			// WARNING: messages are send as MIME type "text/event-stream", which is defined as
			// "Each message is sent as a block of text terminated by a pair of newlines. "
			//https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events
			//_, err := fmt.Fprintf(w, "event: time\ndata: <div sse-swap='time'>%s</div>\n\n", data)
			// FIXME: There seems to be a 64K limit. Probably at the go-sse receiver side
			n, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
				sseMsg.EventType, sseMsg.Payload)
			_ = n
			//_, err := fmt.Fprint(w, sseMsg)
			if err != nil {
				slog.Error("Error writing event", "event", sseMsg.EventType,
					"size", len(sseMsg.Payload))
			}
			w.(http.Flusher).Flush()
			break
		case <-r.Context().Done(): // remote client connection closed
			slog.Info("Remote client disconnected (read context)")
			//close(sseChan)
			done = true
			break
		}
	}
	cs.DeleteSSEChan(sseChan)
	// TODO: if all connections are closed for this client send a disconnected event

	slog.Info("SseHandler: sse connection closed",
		slog.String("remote", r.RemoteAddr),
		slog.String("clientID", cs.GetClientID()),
		slog.Int("nr sse connections", cs.GetNrConnections()),
	)
}