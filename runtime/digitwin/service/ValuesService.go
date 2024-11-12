package service

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/digitwin/store"
)

// ValuesService provides access to digital thing values by consumers
type ValuesService struct {
	// underlying store for the digital twin objects
	dtwStore *store.DigitwinStore
}

// QueryAction returns the current status of the action
func (svc *ValuesService) QueryAction(
	consumerID string, args digitwin.ValuesQueryActionArgs) (v digitwin.ActionValue, err error) {

	return svc.dtwStore.QueryAction(args.ThingID, args.Name)
}

// QueryAllActions returns the list of the latest actions on the thing
func (svc *ValuesService) QueryAllActions(
	consumerID string, dThingID string) ([]digitwin.ActionValue, error) {

	actionMap, err := svc.dtwStore.QueryAllActions(dThingID)
	return utils.Map2Array(actionMap), err
}

// ReadAllEvents returns a list of known digitwin instance event values
func (svc *ValuesService) ReadAllEvents(
	consumerID string, dThingID string) ([]digitwin.ThingValue, error) {

	evMap, err := svc.dtwStore.ReadAllEvents(dThingID)
	return utils.Map2Array(evMap), err
}

// ReadAllProperties returns a map of known digitwin instance property values
func (svc *ValuesService) ReadAllProperties(
	consumerID string, dThingID string) ([]digitwin.ThingValue, error) {

	propMap, err := svc.dtwStore.ReadAllProperties(dThingID)
	return utils.Map2Array(propMap), err
}

// ReadEvent returns the latest event of a digitwin instance
func (svc *ValuesService) ReadEvent(
	consumerID string, args digitwin.ValuesReadEventArgs) (digitwin.ThingValue, error) {

	return svc.dtwStore.ReadEvent(args.ThingID, args.Name)
}

// ReadProperty returns the last known property value of the given name,
// or an empty value if no value is known.
// This returns an error if the dThingID doesn't exist.
func (svc *ValuesService) ReadProperty(
	consumerID string, args digitwin.ValuesReadPropertyArgs) (p digitwin.ThingValue, err error) {

	return svc.dtwStore.ReadProperty(args.ThingID, args.Name)
}

func NewDigitwinValuesService(dtwStore *store.DigitwinStore) *ValuesService {
	svc := &ValuesService{
		dtwStore: dtwStore,
	}
	return svc
}
