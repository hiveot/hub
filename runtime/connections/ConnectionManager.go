package connections

import (
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"slices"
	"sync"
)

// ConnectionManager tracks existing connections through their connection IDs.
//
// Each protocol binding session can have zero or more connections per user.
// Connection ID's are used to differentiate between multiple connections from the
// same client.
//
// The connection manager support a lookup of connections by connection ID, sessionID
// and by clientID, where a clientID is the account ID of the client.
//
// Note: The SSE-SC protocol binding event and property subscriptions take place
// out-of-band from the connection itself (using http posts). To link the subscription request
// to the connection, the http binding expects a connection-ID in the request header.
// This problem is specific to the http binding and not a concern of this connection manager.
type ConnectionManager struct {
	// connections by connection-ID
	cidConnections map[string]IClientConnection

	// connection IDs by clientID
	clientConnections map[string][]string

	// mutex to manage the connections
	mux sync.RWMutex

	// Session manager
	//sm *SessionManager
}

// AddConnection adds a new connection.
// This requires the connection to have a unique connection ID (cid).
// If an endpoint with this cid exists both connections are forcibly closed
// and an error is returned.
func (cm *ConnectionManager) AddConnection(c IClientConnection) error {
	cm.mux.Lock()
	defer cm.mux.Unlock()

	cid := c.GetConnectionID()
	clientID := c.GetClientID()

	// Refuse this if an existing connection with this ID exist
	existingConn, _ := cm.cidConnections[cid]
	if existingConn != nil {
		err := fmt.Errorf("AddConnection. The connection ID '%s' of client '%s' already exists",
			cid, existingConn.GetClientID())
		slog.Error("AddConnection: duplicate CID", "err", err.Error())
		existingConn.Close()
		c.Close()
		go cm.RemoveConnection(cid)
		return err
	}
	cm.cidConnections[cid] = c
	// update the client index
	clientList := cm.clientConnections[clientID]
	if clientList == nil {
		clientList = []string{cid}
	} else {
		clientList = append(clientList, cid)
	}
	cm.clientConnections[clientID] = clientList
	return nil
}

// CloseAllClientConnections closes all connections of the given client.
func (cm *ConnectionManager) CloseAllClientConnections(clientID string) {
	cm.mux.Lock()
	defer cm.mux.Unlock()

	cList := cm.clientConnections[clientID]
	for _, cid := range cList {
		// force-close the connection
		c := cm.cidConnections[cid]
		if c != nil {
			delete(cm.cidConnections, cid)
			c.Close()
		}
	}
	delete(cm.clientConnections, clientID)
}

// CloseAll force-closes all connections
func (cm *ConnectionManager) CloseAll() {
	cm.mux.Lock()
	defer cm.mux.Unlock()

	slog.Info("RemoveAll. Closing remaining connections", "count", len(cm.cidConnections))
	for cid, c := range cm.cidConnections {
		_ = cid
		c.Close()
	}
	cm.cidConnections = make(map[string]IClientConnection)
	cm.clientConnections = make(map[string][]string)
}

// ForEachConnection invoke handler for each client connection
// Intended for publishing event and property updates to subscribers
func (cm *ConnectionManager) ForEachConnection(handler func(c IClientConnection)) {
	// collect a list of connections
	cm.mux.Lock()
	connList := make([]IClientConnection, 0, len(cm.clientConnections))
	for _, c := range cm.cidConnections {
		connList = append(connList, c)
	}
	cm.mux.Unlock()
	for _, c := range connList {
		// TODO: TBD pros/cons for running this in the background vs synchronously?
		handler(c)
	}
}

// GetConnectionByCID locates the connection of the client using its connectionID
// This returns nil if no connection was found with the given cid
func (cm *ConnectionManager) GetConnectionByCID(cid string) (c IClientConnection) {

	cm.mux.Lock()
	defer cm.mux.Unlock()
	c = cm.cidConnections[cid]
	return c
}

// GetConnectionByClientID locates the first connection of the client using its account ID.
// Intended to find agents which only have a single connection.
// This returns nil if no connection was found with the given login
func (cm *ConnectionManager) GetConnectionByClientID(clientID string) (c IClientConnection) {

	cm.mux.Lock()
	defer cm.mux.Unlock()
	cList := cm.clientConnections[clientID]
	if len(cList) == 0 {
		return nil
	}
	// return the first connection of this client
	c = cm.cidConnections[cList[0]]
	if c == nil {
		slog.Error("GetConnectionByClientID: the client's connection list has disconnected endpoints",
			"clientID", clientID, "nr alleged connections", len(cList))
	}
	return c
}

// PublishEvent broadcasts an event message to subscribers of this event.
func (cm *ConnectionManager) PublishEvent(
	dThingID string, name string, value any, messageID string, agentID string) {

	slog.Info("PublishEvent (to subscribers)",
		slog.String("dThingID", dThingID),
		slog.String("name", name),
		slog.String("agentID", agentID),
	)
	cm.ForEachConnection(func(c IClientConnection) {
		c.PublishEvent(dThingID, name, value, messageID, agentID)
	})
}

// PublishProperty broadcasts a property update to subscribers of this event.
func (cm *ConnectionManager) PublishProperty(
	dThingID string, name string, value any, messageID string, agentID string) {

	slog.Info("PublishProperty (to subscribers)",
		slog.String("dThingID", dThingID),
		slog.String("name", name),
		slog.String("agentID", agentID),
	)
	cm.ForEachConnection(func(c IClientConnection) {
		c.PublishProperty(dThingID, name, value, messageID, agentID)
	})
}

// RemoveConnection removes the connection by its connectionID
// This will close the connnection if it isn't closed already.
// Call this after the connection is closed or before closing.
func (cm *ConnectionManager) RemoveConnection(cid string) {
	cm.mux.Lock()
	defer cm.mux.Unlock()

	var clientID = ""
	existingConn := cm.cidConnections[cid]
	// force close the existing connection just in case
	if existingConn != nil {
		clientID = existingConn.GetClientID()
		existingConn.Close()
		delete(cm.cidConnections, cid)
	}

	// remove the cid from the client connection list
	if clientID == "" {
		slog.Error("RemoveConnection: existing connection has no clientID", "cid", cid)
		return
	}
	clientCids := cm.clientConnections[clientID]
	i := slices.Index(clientCids, cid)
	if i < 0 {
		slog.Error("RemoveConnection: existing connection not in client's cid list but is should have been",
			"clientID", clientID, "cid", cid)

		// TODO: considering the impact of this going wrong, is it better to recover?
		// A: delete the bad entry and try the next connection
		// B: close all client connections

	} else {
		clientCids = utils.Remove(clientCids, i)
		cm.clientConnections[clientID] = clientCids
	}
}

// NewConnectionManager creates a new instance of the connection manager
func NewConnectionManager() *ConnectionManager {

	cm := &ConnectionManager{
		cidConnections:    make(map[string]IClientConnection),
		clientConnections: make(map[string][]string),
		mux:               sync.RWMutex{},
	}
	return cm
}
