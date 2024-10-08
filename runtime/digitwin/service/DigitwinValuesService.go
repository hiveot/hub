package service

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/utils"
)

// DigitwinValuesService manages reading digital thing values
type DigitwinValuesService struct {
	// underlying store for the digital twin objects
	dtwStore *DigitwinStore
}

// ReadAction returns the last known action invocation status of the given name
func (svc *DigitwinValuesService) ReadAction(
	consumerID string, args digitwin.ValuesReadActionArgs) (v digitwin.ActionValue, err error) {

	return svc.dtwStore.ReadAction(args.ThingID, args.Name)
}

// ReadAllActions returns the map of the latest actions on the thing
func (svc *DigitwinValuesService) ReadAllActions(
	consumerID string, dThingID string) ([]digitwin.ActionValue, error) {

	actionMap, err := svc.dtwStore.ReadAllActions(dThingID)
	return utils.Map2Array(actionMap), err
}

// ReadAllEvents returns a list of known digitwin instance event values
func (svc *DigitwinValuesService) ReadAllEvents(
	consumerID string, dThingID string) ([]digitwin.EventValue, error) {

	evMap, err := svc.dtwStore.ReadAllEvents(dThingID)
	return utils.Map2Array(evMap), err
}

// ReadAllProperties returns a map of known digitwin instance property values
func (svc *DigitwinValuesService) ReadAllProperties(
	consumerID string, dThingID string) ([]digitwin.PropertyValue, error) {

	propMap, err := svc.dtwStore.ReadAllProperties(dThingID)
	return utils.Map2Array(propMap), err
}

// ReadEvent returns the latest event of a digitwin instance
func (svc *DigitwinValuesService) ReadEvent(
	consumerID string, args digitwin.ValuesReadEventArgs) (digitwin.EventValue, error) {

	return svc.dtwStore.ReadEvent(args.ThingID, args.Name)
}

// ReadProperty returns the last known property value of the given name,
// or an empty value if no value is known.
// This returns an error if the dThingID doesn't exist.
func (svc *DigitwinValuesService) ReadProperty(
	consumerID string, args digitwin.ValuesReadPropertyArgs) (p digitwin.PropertyValue, err error) {

	return svc.dtwStore.ReadProperty(args.ThingID, args.Name)
}

func NewDigitwinValuesService(dtwStore *DigitwinStore) *DigitwinValuesService {
	svc := &DigitwinValuesService{
		dtwStore: dtwStore,
	}
	return svc
}
