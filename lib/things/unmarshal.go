package things

import "encoding/json"

// Helper methods for unmarshalling a TD, TD list, ThingMessage, and its map instance

// UnmarshalTD unmarshals a JSON encoded TD
func UnmarshalTD(tdJSON string) (td *TD, err error) {
	td = &TD{}
	err = json.Unmarshal([]byte(tdJSON), td)
	return td, err
}

func UnmarshalTDList(tdListJSON []string) (tdList []*TD, err error) {
	tdList = make([]*TD, 0, len(tdListJSON))
	for _, tdJson := range tdListJSON {
		td := TD{}
		err = json.Unmarshal([]byte(tdJson), &td)
		if err == nil {
			tdList = append(tdList, &td)
		}
	}
	return tdList, err
}

func UnmarshalThingValue(tmJSON string) (tm ThingMessage, err error) {
	err = json.Unmarshal([]byte(tmJSON), &tm)
	return tm, err

}

func UnmarshalThingValueMap(tmMapJSON string) (tmm ThingMessageMap, err error) {
	err = json.Unmarshal([]byte(tmMapJSON), &tmm)
	return tmm, err
}
