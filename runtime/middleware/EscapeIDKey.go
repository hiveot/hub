package middleware

import (
	"github.com/hiveot/hub/lib/things"
	"strings"
)

// EscapeIDKey replaces spaces in thing ID and keys with dash
// NOTE that TD documents are not escaped. See the directory handler.
func EscapeIDKey(msg *things.ThingMessage) (*things.ThingMessage, error) {
	msg.ThingID = strings.ReplaceAll(msg.ThingID, " ", "-")
	msg.Key = strings.ReplaceAll(msg.Key, " ", "-")
	return msg, nil
}
