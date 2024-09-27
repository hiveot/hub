package sseserver

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
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
		slog.Warn("No session available yet, telling client to delay retry to 10 seconds")

		// set retry to a large number
		// while this doesn't redirect, it does stop it from holding a connection.
		// see https://javascript.info/server-sent-events#reconnection
		//_, _ = fmt.Fprintf(w, "retry: %s\nevent:%s\n\n",
		//	"30000", "logout")
		//w.WriteHeader(http.StatusUnauthorized)

		// NOTE: the above works while the below does not???
		errMsg := fmt.Sprintf("retry: %s\nevent:%s\n\n",
			"10000", "logout")
		http.Error(w, errMsg, http.StatusUnauthorized)
		//w.Write([]byte(errMsg))
		w.(http.Flusher).Flush()
		return
	}

	// establish a client event channel for sending messages back to the client
	sseChan := cs.CreateSSEChan()

	// Send a ping event as the go-sse client doesn't have a 'connected callback'
	pingEvent := sessions.SSEEvent{EventType: hubclient.PingMessage}
	sseChan <- pingEvent

	slog.Info("SseHandler. New SSE connection",
		slog.String("RemoteAddr", r.RemoteAddr),
		slog.String("clientID", cs.GetClientID()),
		slog.String("protocol", r.Proto),
		slog.String("sessionID", cs.GetSessionID()),
		slog.Int("nr sse connections", cs.GetNrConnections()),
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
			go cs.CloseSSEChan(sseChan)
		}
	}()

	// read the message channel until it closes
	for !done { // sseMsg := range sseChan {
		select {
		case sseMsg, ok := <-sseChan: // received event
			var err error

			if !ok { // channel was closed by session
				done = true
				// ending the read loop and returning will close the connection
				break
			}
			slog.Debug("SseHandler: sending sse event to client",
				slog.String("sessionID", cs.GetSessionID()),
				slog.String("clientID", cs.GetClientID()),
				slog.String("sse eventType", sseMsg.EventType),
			)
			// write the message with or without messageID
			if sseMsg.ID == "" {
				_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
					sseMsg.EventType, sseMsg.Payload)
			} else {
				_, err = fmt.Fprintf(w, "event: %s\nid:%s\ndata: %s\n\n",
					sseMsg.EventType, sseMsg.ID, sseMsg.Payload)
			}
			if err != nil {
				// the connection might be closed.
				// don't exit the loop until the receive channel is closed.
				// just keep processing the message until that happens
				// closed go channels panic when written to. So keep reading.
				slog.Error("Error writing SSE event", "ID", sseMsg.ID,
					"size", len(sseMsg.Payload))
			}
			w.(http.Flusher).Flush()
		}
	}
	//cs.DeleteSSEChan(sseChan)
	slog.Info("SseHandler: sse connection closed",
		slog.String("remote", r.RemoteAddr),
		slog.String("clientID", cs.GetClientID()),
		slog.Int("nr sse connections", cs.GetNrConnections()),
	)
}
