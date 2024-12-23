package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/wot"
	"net/http"
)

// HandleGenericHttpOp is the generic HRef handler that retrieves the operation,
// thingID, and name from the URL.
// The downside is that this doesn't quite follow the REST paradigm of addressing
// resources using HTTP methods.
func (svc *HttpTransportServer) HandleGenericHttpOp(w http.ResponseWriter, r *http.Request) {

	// NOTE: THIS IS EXPERIMENTAL
	op := chi.URLParam(r, "operation")
	// responses have a requestID
	//if op == "" {
	//	slog.Error("HandleGenericHttpOp: missing operation")
	//	return
	//}

	// the operation determines if this is a get or post
	if op == wot.OpInvokeAction ||
		op == wot.OpWriteProperty || op == wot.OpWriteMultipleProperties ||
		op == wot.OpReadProperty || op == wot.OpReadMultipleProperties ||
		op == wot.HTOpReadEvent || op == wot.HTOpReadAllEvents ||
		op == wot.HTOpReadTD ||
		op == wot.OpQueryAction || op == wot.OpQueryAllActions {
		svc._handleRequestMessage(op, w, r)
	} else {
		svc._handleNotification(op, w, r)
	}
}
