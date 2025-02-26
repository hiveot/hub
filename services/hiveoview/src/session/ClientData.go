package session

import (
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	jsoniter "github.com/json-iterator/go"
	"sync"
)

// SessionData containing the persisted client data, UI preferences, and
// dashboard configurations.
type SessionData struct {
	mux sync.RWMutex // mutex to protect the maps below

	// client dashboard(s) - allow for serialization. do not use directly
	Dashboards map[string]*DashboardModel `json:"dashboards"`

	// storage bucket of client state
	dataBucket buckets.IBucket

	// UI preferences ...
}

// DeleteDashboard removes a dashboard from the model
func (model *SessionData) DeleteDashboard(id string) {
	model.mux.Lock()
	delete(model.Dashboards, id)
	model.mux.Unlock()

	_ = model.SaveState()
}

// GetDashboard returns a dashboard with the given ID
func (model *SessionData) GetDashboard(id string) (d DashboardModel, found bool) {
	model.mux.RLock()
	defer model.mux.RUnlock()
	dashboard, found := model.Dashboards[id]
	if found {
		if dashboard.GridLayouts == nil {
			dashboard.GridLayouts = make(map[string]string)
		}
		return *dashboard, found
	}
	return
}

// GetFirstDashboard returns the first dashboard in the map
// This returns a new empty dashboard if there are no dashboards
func (model *SessionData) GetFirstDashboard() (d DashboardModel) {
	model.mux.RLock()
	defer model.mux.RUnlock()
	if len(model.Dashboards) == 0 {
		d = NewDashboard("default", "New Dashboard")
	}
	for _, first := range model.Dashboards {
		d = *first
		break
	}
	return d
}

// LoadState loads the client session state containing dashboard and other model data,
// and clear 'clientModelChanged' status
func (model *SessionData) LoadState() error {
	dashboards := make(map[string]*DashboardModel)

	// load the stored view state from the state service
	dashboardsRaw, err := model.dataBucket.Get(dashboardsStorageKey)
	// nothing saved so use defaults
	if err != nil {
		defaultDashboard := NewDashboard("default", "New Dashboard")
		dashboards["default"] = &defaultDashboard
		err = nil
	} else {
		err = jsoniter.Unmarshal(dashboardsRaw, &dashboards)
		if err != nil {
			err = fmt.Errorf("invalid dashboard data in store: %w", err)
		}
	}

	// then lock and load
	model.mux.Lock()
	defer model.mux.Unlock()
	model.Dashboards = dashboards
	return nil
}

// SaveState stores the current client session model.
// if 'clientModelChanged' is set.
//
// This returns an error if the state service is not reachable.
func (model *SessionData) SaveState() error {
	model.mux.RLock()
	dashboardsRaw, _ := jsoniter.Marshal(model.Dashboards)
	model.mux.RUnlock()

	err := model.dataBucket.Set(dashboardsStorageKey, dashboardsRaw)
	return err
}

// UpdateDashboard adds or replaces a dashboard in the model
func (model *SessionData) UpdateDashboard(dashboard *DashboardModel) {
	model.mux.Lock()
	model.Dashboards[dashboard.ID] = dashboard
	model.mux.Unlock()
	_ = model.SaveState()
}

// GetTile returns a tile in the model with a flag if it was found
//func (model *SessionData) GetTile(id string) (tile DashboardTile, found bool) {
//	model.mux.RLock()
//	defer model.mux.RUnlock()
//
//	tileRef, found := model.Tiles[id]
//	if found {
//		tile = *tileRef
//	}
//	return tile, found
//}

// UpdateTile adds or replaces a tile in the model
//func (model *SessionData) UpdateTile(tile *DashboardTile) {
//	model.mux.Lock()
//	defer model.mux.Unlock()
//
//	model.Tiles[tile.ID] = tile
//	model.dataChanged = true
//}

func NewClientDataModel(dataBucket buckets.IBucket) *SessionData {
	model := SessionData{
		mux:        sync.RWMutex{},
		dataBucket: dataBucket,
		Dashboards: make(map[string]*DashboardModel),
	}
	_ = model.LoadState()
	return &model
}
