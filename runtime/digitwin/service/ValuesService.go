package service

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/digitwin/store"
	"github.com/hiveot/hub/transports/tputils"
)

// ValuesService provides access to digital thing values by consumers
type ValuesService struct {
	// underlying store for the digital twin objects
	dtwStore *store.DigitwinStore
}

// QueryAction returns the current status of the action
func (svc *ValuesService) QueryAction(clientID string,
	args digitwin.ValuesQueryActionArgs) (av digitwin.ActionStatus, err error) {
	//convert action status to action value, because ... need generated agent code
	as, err := svc.dtwStore.QueryAction(args.ThingID, args.Name)
	if err == nil {
		err = tputils.Decode(as, &av)
	}
	return av, err
}

// ReadAllEvents returns a list of known digitwin instance event values
func (svc *ValuesService) ReadAllEvents(clientID string,
	dThingID string) ([]digitwin.ThingValue, error) {

	evMap, err := svc.dtwStore.ReadAllEvents(dThingID)
	return utils.Map2Array(evMap), err
}

// ReadAllProperties returns a map of known digitwin instance property values
func (svc *ValuesService) ReadAllProperties(clientID string,
	dThingID string) ([]digitwin.ThingValue, error) {

	propMap, err := svc.dtwStore.ReadAllProperties(dThingID)
	return utils.Map2Array(propMap), err
}

// ReadEvent returns the latest event of a digitwin instance
func (svc *ValuesService) ReadEvent(clientID string,
	args digitwin.ValuesReadEventArgs) (digitwin.ThingValue, error) {

	return svc.dtwStore.ReadEvent(args.ThingID, args.Name)
}

// ReadProperty returns the last known property value of the given name,
// or an empty value if no value is known.
// This returns an error if the dThingID doesn't exist.
func (svc *ValuesService) ReadProperty(clientID string,
	args digitwin.ValuesReadPropertyArgs) (p digitwin.ThingValue, err error) {

	return svc.dtwStore.ReadProperty(args.ThingID, args.Name)
}

func NewDigitwinValuesService(dtwStore *store.DigitwinStore) *ValuesService {
	svc := &ValuesService{
		dtwStore: dtwStore,
	}
	return svc
}
