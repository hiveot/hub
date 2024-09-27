// Package service with digital twin directory functions
package service

import (
	"encoding/json"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
)

// MakeDigtalTwin returns the digital twin from a agent provided TD
func (svc *DigitwinService) MakeDigtalTwin(agentID string, td tdd.TD) (dtd tdd.TD, err error) {

	dtd = td
	dtd.ID = tdd.MakeDigiTwinThingID(agentID, td.ID)
	if svc.tb != nil {
		svc.tb.AddTDForms(&dtd)
	}
	return dtd, err
}

// RemoveThing removes a digital thing
func (svc *DigitwinService) RemoveThing(consumerID string, dThingID string) error {
	err := svc.dtwStore.RemoveDTW(dThingID)
	return err
}

// UpdateTD updates the digitwin TD from an agent supplied TD
// This transforms the given TD into the digital twin instance.
func (svc *DigitwinService) UpdateTD(agentID string, thingID string, tdJson string) error {
	slog.Info("UpdateTD")

	var thingTD tdd.TD
	err := json.Unmarshal([]byte(tdJson), &thingTD)
	if err != nil {
		return err
	}
	dtwTD, err := svc.MakeDigtalTwin(agentID, thingTD)
	if err == nil {
		svc.dtwStore.UpdateThing(agentID, thingTD, dtwTD)
	}
	return err
}
