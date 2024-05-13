package httpsbinding

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/outbox"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"net/http"
)

// Experimental HTTP API
// A convenience api to get the directory and values of things

// HandleGetEvents returns a list of latest messages from a Thing
// Parameters: thingID
func (svc *HttpsBinding) HandleGetEvents(w http.ResponseWriter, r *http.Request) {
	cs, thingID, key, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// this request can simply be turned into an action message.
	args := outbox.ReadLatestArgs{ThingID: thingID}
	// just a single key supported
	if key != "" {
		args.Keys = []string{key}
	}
	argsJSON, _ := json.Marshal(args)
	msg := things.NewThingMessage(
		vocab.MessageTypeAction, outbox.ThingID, outbox.ReadLatestMethod, argsJSON, cs.GetClientID())
	stat := svc.handleMessage(msg)
	resp := outbox.ReadLatestResp{}
	// re-serialize just the ThingMessageMap for the response
	err = json.Unmarshal(stat.Reply, &resp)
	var replyJSON []byte
	if err == nil {
		replyJSON, err = json.Marshal(resp.Values)
	}
	svc.writeReply(w, replyJSON, err)
}

// HandleGetThings returns a list of things in the directory
// No parameters
func (svc *HttpsBinding) HandleGetThings(w http.ResponseWriter, r *http.Request) {
}
