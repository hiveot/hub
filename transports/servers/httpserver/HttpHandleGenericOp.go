package httpserver

import (
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/httpserver/httpcontext"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/http"
)

// HandleGenericHttpOp is the generic HRef handler that retrieves the operation,
// thingID, and name from the URL.
func (svc *HttpTransportServer) HandleGenericHttpOp(w http.ResponseWriter, r *http.Request) {

	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if rp.Op == "" {
		slog.Error("HandleGenericHttpOp: missing operation")
		return
	}
	// pass the event to the digitwin service for further processing
	requestID := r.Header.Get(transports.RequestIDHeader)
	if requestID == "" {
		requestID = shortid.MustGenerate()
	}
	msg := transports.NewThingMessage(rp.Op, rp.ThingID, rp.Name, rp.Data, rp.ClientID)

	// the operation determines if this is a get or post
	// TODO
	_ = msg
}
