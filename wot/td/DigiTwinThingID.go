package td

import (
	"fmt"
	"strings"
)

const DTWPrefix = "dtw"

// SplitDigiTwinThingID splits the virtual ThingID into the agent ID and native thingID.
// If the thingID does not contain an agentID then the is returned as thingID and
// agentID will be empty.
//
//	dtThingID is the digital twin's thingID that contains the agent's ID
func SplitDigiTwinThingID(dtThingID string) (agentID string, thingID string) {
	// "dtw:agentID:" was prepended to the original thingID
	parts := strings.Split(dtThingID, ":")
	if parts[0] != DTWPrefix && len(parts) < 3 {
		return "", dtThingID
	}
	agentID = parts[1]
	thingID = strings.Join(parts[2:], ":")
	return agentID, thingID
}

// MakeDigiTwinThingID returns the thingID that represents the digital twin Thing
// This is constructed  as: "dtw:{agentID}:{thingID}"
func MakeDigiTwinThingID(agentID string, thingID string) string {
	dtThingID := fmt.Sprintf("dtw:%s:%s", agentID, thingID)
	return dtThingID
}
