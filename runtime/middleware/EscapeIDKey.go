package middleware

import (
	"github.com/hiveot/hub/lib/hubclient"
	"strings"
)

// EscapeIDKey replaces spaces in thing ID and keys with dash
// NOTE that TD documents are not escaped. See the directory handler.
func EscapeIDKey(msg *hubclient.ThingMessage) (*hubclient.ThingMessage, error) {
	msg.ThingID = strings.ReplaceAll(msg.ThingID, " ", "-")
	msg.Name = strings.ReplaceAll(msg.Name, " ", "-")
	return msg, nil
}
