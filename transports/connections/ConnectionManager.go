package connections

import (
	"fmt"
	"github.com/hiveot/hub/transports"
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
	// connections by clcid = {clientID}:{connectionID}
	connectionsByConnectionID map[string]transports.IServerConnection

	// connectionIDs by clientID
	connectionsByClientID map[string][]string

	// mutex to manage the connections
	mux sync.RWMutex
}

// AddConnection adds a new connection.
// This requires the connection to have a unique client connection ID (connectionID).
// If an endpoint with this connectionID exists both connections are forcibly closed
// and an error is returned.
func (cm *ConnectionManager) AddConnection(c transports.IServerConnection) error {
	cm.mux.Lock()
	defer cm.mux.Unlock()

	connectionID := c.GetConnectionID()
	clientID := c.GetClientID()
	// the client's connectionID for lookup
	clcid := clientID + ":" + connectionID

	// Refuse this if an existing connection with this ID exist
	existingConn, _ := cm.connectionsByConnectionID[clcid]
	if existingConn != nil {
		err := fmt.Errorf("AddConnection. The connection ID '%s' of client '%s' already exists",
			connectionID, existingConn.GetClientID())
		slog.Error("AddConnection: duplicate ConnectionID", "connectionID", connectionID, "err", err.Error())
		existingConn.Disconnect()
		c.Disconnect()
		go cm.RemoveConnection(existingConn)
		return err
	}
	cm.connectionsByConnectionID[clcid] = c
	// update the client index
	clientList := cm.connectionsByClientID[clientID]
	if clientList == nil {
		clientList = []string{connectionID}
	} else {
		clientList = append(clientList, connectionID)
	}
	cm.connectionsByClientID[clientID] = clientList
	return nil
}

// CloseAllClientConnections closes all connections of the given client.
func (cm *ConnectionManager) CloseAllClientConnections(clientID string) {
	cm.mux.Lock()
	defer cm.mux.Unlock()

	cList := cm.connectionsByClientID[clientID]
	for _, cid := range cList {
		// force-close the connection
		clcid := clientID + ":" + cid
		c := cm.connectionsByConnectionID[clcid]
		if c != nil {
			delete(cm.connectionsByConnectionID, clcid)
			c.Disconnect()
		}
	}
	delete(cm.connectionsByClientID, clientID)
}

// CloseAll force-closes all connections
func (cm *ConnectionManager) CloseAll() {
	cm.mux.Lock()
	defer cm.mux.Unlock()

	slog.Info("RemoveAll. Closing remaining connections", "count", len(cm.connectionsByConnectionID))
	for clcid, c := range cm.connectionsByConnectionID {
		_ = clcid
		c.Disconnect()
	}
	cm.connectionsByConnectionID = make(map[string]transports.IServerConnection)
	cm.connectionsByClientID = make(map[string][]string)
}

// ForEachConnection invoke handler for each client connection
// Intended for publishing event and property updates to subscribers
// This is concurrent safe as the iteration takes place on a copy
func (cm *ConnectionManager) ForEachConnection(handler func(c transports.IServerConnection)) {
	// collect a list of connections
	cm.mux.Lock()
	connList := make([]transports.IServerConnection, 0, len(cm.connectionsByClientID))
	for _, c := range cm.connectionsByConnectionID {
		connList = append(connList, c)
	}
	cm.mux.Unlock()
	//
	for _, c := range connList {
		// TODO: TBD pros/cons for running this in the background vs synchronously?
		handler(c)
	}
}

// GetConnectionByConnectionID locates the connection of the client using the client's connectionID
// This returns nil if no connection was found with the given connectionID
func (cm *ConnectionManager) GetConnectionByConnectionID(clientID, connectionID string) (c transports.IServerConnection) {

	clcid := clientID + ":" + connectionID
	cm.mux.Lock()
	defer cm.mux.Unlock()
	c = cm.connectionsByConnectionID[clcid]
	return c
}

// GetConnectionByClientID locates the first connection of the client using its account ID.
// Intended to find agents which only have a single connection.
// This returns nil if no connection was found with the given login
func (cm *ConnectionManager) GetConnectionByClientID(clientID string) (c transports.IServerConnection) {

	cm.mux.Lock()
	defer cm.mux.Unlock()
	cList := cm.connectionsByClientID[clientID]
	if len(cList) == 0 {
		return nil
	}
	clcid := clientID + ":" + cList[0]

	// return the first connection of this client
	c = cm.connectionsByConnectionID[clcid]
	if c == nil {
		slog.Error("GetConnectionByClientID: the client's connection list has disconnected endpoints",
			"clientID", clientID, "nr alleged connections", len(cList))
	}
	return c
}

// GetNrConnections returns the number of client connections and nr of unique clients
func (cm *ConnectionManager) GetNrConnections() (int, int) {
	cm.mux.RLock()
	defer cm.mux.RUnlock()
	return len(cm.connectionsByConnectionID), len(cm.connectionsByClientID)
}

// SendNotification sends a response notification to subscribers
// For each subscriber, the correlationID of the subscription is used.
func (cm *ConnectionManager) SendNotification(resp *transports.ResponseMessage) error {

	slog.Debug("SendNotification (to subscribers/observers)",
		slog.String("Operation", resp.Operation),
		slog.String("dThingID", resp.ThingID),
		slog.String("name", resp.Name),
		slog.Any("output", resp.Output),
	)
	// is determined by the server (like MQTT)
	cm.ForEachConnection(func(c transports.IServerConnection) {
		c.SendNotification(*resp)
	})
	return nil
}

// RemoveConnection removes the connection by its connectionID
// This will close the connnection if it isn't closed already.
// Call this after the connection is closed or before closing.
func (cm *ConnectionManager) RemoveConnection(c transports.IServerConnection) {
	cm.mux.Lock()
	defer cm.mux.Unlock()

	clientID := c.GetClientID()
	connectionID := c.GetConnectionID()
	clcid := clientID + ":" + connectionID
	existingConn := cm.connectionsByConnectionID[clcid]
	// force close the existing connection just in case
	if existingConn != nil {
		//clientID = existingConn.GetClientID()
		existingConn.Disconnect()
		delete(cm.connectionsByConnectionID, clcid)
	} else if len(cm.connectionsByConnectionID) > 0 {
		// this is unexpected. Not all connections were closed but this one is gone.
		slog.Warn("RemoveConnection: connectionID not found",
			"clcid", clcid)
		return
	}
	// remove the cid from the client connection list
	clientCids := cm.connectionsByClientID[clientID]
	i := slices.Index(clientCids, connectionID)
	if i < 0 {
		slog.Info("RemoveConnection: existing connection not in the connectionID list. Was it forcefully removed?",
			"clientID", clientID, "connectionID", connectionID)

		// TODO: considering the impact of this going wrong, is it better to recover?
		// A: delete the bad entry and try the next connection
		// B: close all client connections

	} else {
		clientCids = slices.Delete(clientCids, i, i+1)
		//clientCids = utils.Remove(clientCids, i)
		cm.connectionsByClientID[clientID] = clientCids
	}
}

// NewConnectionManager creates a new instance of the connection manager
func NewConnectionManager() *ConnectionManager {

	cm := &ConnectionManager{
		connectionsByConnectionID: make(map[string]transports.IServerConnection),
		connectionsByClientID:     make(map[string][]string),
		mux:                       sync.RWMutex{},
	}
	return cm
}
