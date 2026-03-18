package digitwin

import (
	"fmt"
	"strings"

	digitwinapi "github.com/hiveot/hivekit/go/modules/digitwin/api"
)

// Create a digital twin ID from the agent and device thingID
// this joins the digitwin prefix with the agent and thingID
func MakeDigitwinID(agentID string, thingID string) string {
	digitwinThingID := fmt.Sprintf("%s%s:%s",
		digitwinapi.DigitwinIDPrefix, agentID, thingID)
	return digitwinThingID
}

// Split the digital twin ID into the agent and device thingID
// This returns an error if the given ID is not a digitwin ID
func SplitDigitwinID(digitwinID string) (agentID string, thingID string, err error) {
	parts := strings.Split(digitwinID, ":")
	if len(parts) != 3 || !strings.HasPrefix(digitwinID, digitwinapi.DigitwinIDPrefix) {
		return "", "", fmt.Errorf("The given id '%s' is not a digital twin thingID", digitwinID)
	}
	return parts[1], parts[2], nil
}
