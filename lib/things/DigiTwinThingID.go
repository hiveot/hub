package things

import (
	"fmt"
	"strings"
)

// SplitDigiTwinThingID splits the virtual ThingID into the agent ID and physical thingID.
// If the agent is unknown then this return an empty string and 'found is false
//
//	dtThingID is the digital twin's thingID that contains the agent's ID
func SplitDigiTwinThingID(dtThingID string) (agentID string, thingID string, found bool) {
	// "dtw:agentID:" was prepended to the original thingID
	parts := strings.Split(dtThingID, ":")
	if len(parts) < 3 {
		return "", dtThingID, false
	}
	agentID = parts[1]
	thingID = strings.Join(parts[2:], ":")
	return agentID, thingID, true
}

// MakeDigiTwinThingID returns the thingID that represents the digital twin Thing
// This is constructed  as: "dtw:{agentID}:{thingID}"
func MakeDigiTwinThingID(agentID string, thingID string) string {
	dtThingID := fmt.Sprintf("dtw:%s:%s", agentID, thingID)
	return dtThingID
}
