package httpsbinding

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"io"
	"log/slog"
	"net/http"
)

// getMessageSession reads the message and identifies the sender's session
func (svc *HttpsBinding) getMessageSession(msgType string, r *http.Request) (
	msg *things.ThingMessage, session *ClientSession, err error) {

	// get the required client session of this agent
	ctxSession := r.Context().Value(SessionContextID)
	if ctxSession == nil {
		err = fmt.Errorf("missing session")
		return nil, nil, err
	}
	cs := ctxSession.(*ClientSession)

	// build a message from the URL and payload
	thingID := chi.URLParam(r, "thingID")
	key := chi.URLParam(r, "key")
	data, err := io.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("failed reading message body: %w", err)
	}
	msg = things.NewThingMessage(msgType, thingID, key, data, cs.clientID)
	return msg, cs, err
}

// handlePostAction handles a posted action by a consumer
func (svc *HttpsBinding) handlePostAction(w http.ResponseWriter, r *http.Request) {
	msg, session, err := svc.getMessageSession(vocab.MessageTypeAction, r)
	if err != nil {
		slog.Warn("handlePostAction", "err", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_ = session
	reply, err := svc.handleMessage(msg)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(reply)
	w.WriteHeader(http.StatusOK)
	return
}

// onPostEvent handles a posted event by agent
func (svc *HttpsBinding) handlePostEvent(w http.ResponseWriter, r *http.Request) {
	msg, session, err := svc.getMessageSession(vocab.MessageTypeEvent, r)
	if err != nil {
		slog.Warn("handlePostEvent", "err", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_ = session
	_, err = svc.handleMessage(msg)
	if err != nil {
		slog.Error("handlePostEvent", "err", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

// handlePostRPC handles a rpc request posted by a consumer
func (svc *HttpsBinding) handlePostRPC(w http.ResponseWriter, r *http.Request) {
	msg, session, err := svc.getMessageSession(vocab.MessageTypeAction, r)
	if err != nil {
		slog.Warn("handlePostRPC", "err", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_ = session
	reply, err := svc.handleMessage(msg)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(reply)
	w.WriteHeader(http.StatusOK)
	return
}
