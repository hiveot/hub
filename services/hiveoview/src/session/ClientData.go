package session

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/hiveot/hivekit/go/buckets"
	jsoniter "github.com/json-iterator/go"
)

// SessionData containing the persisted client data, UI preferences, and
// dashboard configurations.
type SessionData struct {
	mux sync.RWMutex // mutex to protect the maps below

	// client dashboard(s) - allow for serialization. do not use directly
	Dashboards []*DashboardModel `json:"dashboards"`

	// presentation order of dashboards
	DashboardIds []string

	// storage bucket of client state
	dataBucket buckets.IBucket

	// UI preferences ...
}

// DeleteDashboard removes a dashboard from the model
func (model *SessionData) DeleteDashboard(id string) {
	model.mux.Lock()
	newList := slices.DeleteFunc(model.Dashboards, func(d *DashboardModel) bool {
		return d.ID == id
	})
	model.Dashboards = newList
	model.mux.Unlock()

	_ = model.SaveState()
}

// GetDashboard returns a dashboard with the given ID
func (model *SessionData) GetDashboard(id string) (d DashboardModel, found bool) {
	model.mux.RLock()
	defer model.mux.RUnlock()
	for _, dashboard := range model.Dashboards {
		if dashboard.ID == id {
			// recover from bad data
			if dashboard.GridLayouts == nil {
				dashboard.GridLayouts = make(map[string]string)
			}
			// in case dashboards have been newly created
			if dashboard.Tiles == nil {
				dashboard.Tiles = make(map[string]DashboardTile)
			}
			// return a copy
			return *dashboard, true
		}
	}
	return d, false
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
	dashboards := make([]*DashboardModel, 0)

	// load the stored view state from the state service
	dashboardsRaw, err := model.dataBucket.Get(dashboardsStorageKey)
	if err == nil {
		err = jsoniter.Unmarshal(dashboardsRaw, &dashboards)
		if err != nil {
			err = fmt.Errorf("invalid dashboard data in store: %w", err)
		}
		if err != nil {
			slog.Error(err.Error())
		}
	}
	// nothing saved so use defaults
	if len(dashboards) == 0 {
		defaultDashboard := NewDashboard("default", "New Dashboard")
		dashboards = append(dashboards, &defaultDashboard)
		err = nil
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
func (model *SessionData) UpdateDashboard(d *DashboardModel) error {
	if d.ID == "" {
		slog.Error("UpdateDashboard: missing ID", "title", d.Title)
		return errors.New("missing dashboard ID")
	}
	found := false
	model.mux.Lock()
	for i, dashboard := range model.Dashboards {
		if dashboard.ID == d.ID {
			model.Dashboards[i] = d
			found = true
			break
		}
	}
	if !found {
		model.Dashboards = append(model.Dashboards, d)
	}
	model.mux.Unlock()
	_ = model.SaveState()
	return nil
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
		Dashboards: make([]*DashboardModel, 0),
	}
	_ = model.LoadState()
	return &model
}
