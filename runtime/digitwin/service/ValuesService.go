package service

import (
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/runtime/digitwin/store"
	"github.com/hiveot/hub/transports/tputils"
)

// ValuesService provides access to digital thing values by consumers
// This implements the IValuesSvcService interface
type ValuesService struct {
	// underlying store for the digital twin objects
	dtwStore *store.DigitwinStore
}

// QueryAction returns the current status of the action
func (svc *ValuesService) QueryAction(clientID string,
	args digitwin.ThingValuesQueryActionArgs) (av digitwin.ActionStatus, err error) {
	//convert action status to action value, because ... need generated agent code
	as, err := svc.dtwStore.QueryAction(args.ThingID, args.Name)
	if err == nil {
		err = tputils.Decode(as, &av)
	}
	return av, err
}

// QueryAllActions returns the current status of all thing actions
func (svc *ValuesService) QueryAllActions(clientID string,
	thingID string) (av map[string]digitwin.ActionStatus, err error) {

	//convert action status to action value, because ... need generated agent code
	as, err := svc.dtwStore.QueryAllActions(thingID)
	if err == nil {
		err = tputils.Decode(as, &av)
	}
	return av, err
}

// ReadAllEvents returns a list of known digitwin instance event values
func (svc *ValuesService) ReadAllEvents(clientID string,
	dThingID string) (map[string]digitwin.ThingValue, error) {

	evMap, err := svc.dtwStore.ReadAllEvents(dThingID)
	return evMap, err
}

// ReadAllProperties returns a map of known digitwin instance property values
func (svc *ValuesService) ReadAllProperties(clientID string,
	dThingID string) (map[string]digitwin.ThingValue, error) {

	propMap, err := svc.dtwStore.ReadAllProperties(dThingID)
	return propMap, err
}

// ReadEvent returns the latest event of a digitwin instance
func (svc *ValuesService) ReadEvent(clientID string,
	args digitwin.ThingValuesReadEventArgs) (digitwin.ThingValue, error) {

	return svc.dtwStore.ReadEvent(args.ThingID, args.Name)
}

// ReadProperty returns the last known property value of the given name,
// or an empty value if no value is known.
// This returns an error if the dThingID doesn't exist.
func (svc *ValuesService) ReadProperty(clientID string,
	args digitwin.ThingValuesReadPropertyArgs) (p digitwin.ThingValue, err error) {

	return svc.dtwStore.ReadProperty(args.ThingID, args.Name)
}

func NewDigitwinValuesService(dtwStore *store.DigitwinStore) *ValuesService {
	svc := &ValuesService{
		dtwStore: dtwStore,
	}
	return svc
}
