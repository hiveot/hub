package old

import (
	"log/slog"
	"net/http"
)

// WriteBadRequest logs and respond with bad request error status code and log error
func (srv *TLSServer) WriteBadRequest(resp http.ResponseWriter, errMsg string) {
	slog.Error(errMsg)
	http.Error(resp, errMsg, http.StatusBadRequest)
}

// WriteInternalError logs and responds with internal server error status code and log error
func (srv *TLSServer) WriteInternalError(resp http.ResponseWriter, errMsg string) {
	slog.Error(errMsg)
	http.Error(resp, errMsg, http.StatusInternalServerError)
}

// WriteNotFound logs and respond with 404 resource not found
func (srv *TLSServer) WriteNotFound(resp http.ResponseWriter, errMsg string) {
	slog.Error(errMsg)
	http.Error(resp, errMsg, http.StatusNotFound)
}

// WriteNotImplemented respond with 501 not implemented
func (srv *TLSServer) WriteNotImplemented(resp http.ResponseWriter, errMsg string) {
	slog.Error(errMsg)
	http.Error(resp, errMsg, http.StatusNotImplemented)
}

// WriteUnauthorized responds with unauthorized (401) status code and log http error
// Use this when login fails
func (srv *TLSServer) WriteUnauthorized(resp http.ResponseWriter, errMsg string) {
	slog.Error(errMsg)
	http.Error(resp, errMsg, http.StatusUnauthorized)
}

// WriteForbidden logs and respond with forbidden (403) code and log http error
// Use this when access a resource without sufficient credentials
func (srv *TLSServer) WriteForbidden(resp http.ResponseWriter, errMsg string) {
	slog.Error(errMsg)
	http.Error(resp, errMsg, http.StatusForbidden)
}
