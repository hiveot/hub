package session

import (
	"sync"
)

// ClientDataModel containing the persisted client data, UI preferences, and
// dashboard configurations.
type ClientDataModel struct {
	mux         sync.RWMutex // mutex to protect the maps below
	dataChanged bool

	// client dashboard(s) - allow for serialization. do not use directly
	Dashboards map[string]*DashboardModel `json:"dashboards"`

	// UI preferences ...
}

// Changed returns whether the persistent data has changed
func (model *ClientDataModel) Changed() bool {
	model.mux.RLock()
	defer model.mux.RUnlock()
	return model.dataChanged
}

// DeleteDashboard removes a dashboard from the model
func (model *ClientDataModel) DeleteDashboard(id string) {
	model.mux.Lock()
	defer model.mux.Unlock()

	delete(model.Dashboards, id)
	model.dataChanged = true
}

// GetDashboard returns a dashboard with the given ID
func (model *ClientDataModel) GetDashboard(id string) (d DashboardModel, found bool) {
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
func (model *ClientDataModel) GetFirstDashboard() (d DashboardModel) {
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

// UpdateDashboard adds or replaces a dashboard in the model
func (model *ClientDataModel) UpdateDashboard(dashboard *DashboardModel) {
	model.mux.Lock()
	defer model.mux.Unlock()
	model.Dashboards[dashboard.ID] = dashboard
	model.dataChanged = true
}

// GetTile returns a tile in the model with a flag if it was found
//func (model *ClientDataModel) GetTile(id string) (tile DashboardTile, found bool) {
//	model.mux.RLock()
//	defer model.mux.RUnlock()
//
//	tileRef, found := model.Tiles[id]
//	if found {
//		tile = *tileRef
//	}
//	return tile, found
//}

// SetChanged sets or clears the 'changed' state of the model.
// Intended to clear it after saving
func (model *ClientDataModel) SetChanged(newValue bool) {
	model.mux.Lock()
	defer model.mux.Unlock()
	model.dataChanged = newValue
}

// UpdateTile adds or replaces a tile in the model
//func (model *ClientDataModel) UpdateTile(tile *DashboardTile) {
//	model.mux.Lock()
//	defer model.mux.Unlock()
//
//	model.Tiles[tile.ID] = tile
//	model.dataChanged = true
//}

func NewClientDataModel() *ClientDataModel {
	model := ClientDataModel{
		mux:         sync.RWMutex{},
		dataChanged: false,
		Dashboards:  make(map[string]*DashboardModel),
	}
	return &model
}
