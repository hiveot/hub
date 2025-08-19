package tlsserver

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/hiveot/hub/messaging"
	jsoniter "github.com/json-iterator/go"
)

// WriteError is a convenience function that logs and writes an error
// If the reply has an error then write a bad request with the error as payload
// This also writes the StatusHeader containing StatusFailed.
func WriteError(w http.ResponseWriter, err error, code int) {
	if code == 0 {
		code = http.StatusBadRequest
	}
	if err != nil {
		slog.Warn("Request error: ", "err", err.Error())
		http.Error(w, err.Error(), code)
	} else {
		//replyHeader := w.Header()
		//replyHeader.Set(StatusHeader, transports.StatusCompleted)
		w.WriteHeader(code)
	}
}

// WriteReply is a convenience function that serializes the data and writes it as a response,
// optionally reporting an error with code BadRequest.
//
// when handled, this returns a 200 status code if no error is returned.
// handled is false means the request is in progress. This returns a 201.
// if an err is returned this returns a 400 bad request or 403 unauthorized error code
// the data can contain error details.
func WriteReply(
	w http.ResponseWriter, handled bool, data any, err error) {
	var payloadJSON string

	if data != nil {
		payloadJSON, _ = jsoniter.MarshalToString(data)
	}
	if err != nil {
		if errors.Is(err, messaging.UnauthorizedError) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	} else if handled {
		if payloadJSON != "" {
			_, _ = w.Write([]byte(payloadJSON))
		}
		// Code 200: https://w3c.github.io/wot-profile/#example-17
		w.WriteHeader(http.StatusOK)
	} else {
		// not handled no error. response will be async
		// Code 201: https://w3c.github.io/wot-profile/#sec-http-sse-profile
		w.WriteHeader(201)
		if payloadJSON != "" {
			_, _ = w.Write([]byte(payloadJSON))
		}
	}
}
