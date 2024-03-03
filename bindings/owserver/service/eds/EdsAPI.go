// Package eds with EDS OWServer API methods
package eds

import (
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// ActuatorTypeVocab maps OWServer names to IoT vocabulary
//var ActuatorTypeVocab = map[string]struct {
//	VocabType string // sensor type from vocabulary
//	Title        string
//	DataType     string
//}{
//	// "BarometricPressureHg": vocab.PropNameAtmosphericPressure, // unit Hg
//	"Relay": {VocabType: vocab.ActionSwitchOff, Title: "Relay", DataType: vocab.WoTDataTypeBool},
//}

// EdsAPI EDS device API properties and methods
type EdsAPI struct {
	address         string     // EDS (IP) address or filename (file://./path/to/name.xml)
	loginName       string     // Basic Auth login name
	password        string     // Basic Auth password
	discoTimeoutSec int        // EDS OWServer discovery timeout
	readMutex       sync.Mutex // prevent concurrent discovery
}

// GetLastAddress returns the last used address of the gateway
// This is either the configured or the discovered address
//func (edsAPI *EdsAPI) GetLastAddress() string {
//	return edsAPI.address
//}

// PollNodes polls the OWServer gateway for nodes and property values
// Returns a list of nodes and a map of device/node ID's containing a map of property name:value
// pairs.
func (edsAPI *EdsAPI) PollNodes() (nodeList []*OneWireNode, err error) {

	// Read the values from the EDS gateway
	if edsAPI.address == "" {
		edsAPI.address, err = Discover(edsAPI.discoTimeoutSec)
		if err != nil {
			return nil, err
		}
	}
	startTime := time.Now()
	rootNode, err := ReadEds(edsAPI.address, edsAPI.loginName, edsAPI.password)
	endTime := time.Now()
	latency := endTime.Sub(startTime)
	if err != nil {
		slog.Error("failed", "err", err)
		return nil, err
	}

	// Extract the nodes and convert properties to vocab names
	nodeList = ParseOneWireNodes(rootNode, latency, true)
	slog.Info("Nodes found", "count", len(nodeList))
	return nodeList, nil
}

// WriteData writes a value to a variable
// this posts a request to devices.html?rom={romID}&variable={variable}&value={value}
func (edsAPI *EdsAPI) WriteData(romID string, variable string, value string) error {
	// TODO: auto config if this is http or https
	writeURL := edsAPI.address + "/devices.htm" +
		"?rom=" + romID + "&variable=" + variable + "&value=" + value
	req, _ := http.NewRequest("GET", writeURL, nil)

	slog.Info("write to OwServer", "URL", writeURL)
	req.SetBasicAuth(edsAPI.loginName, edsAPI.password)
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	_ = resp

	if err != nil {
		slog.Error("Unable to write data to EDS gateway", "err", err.Error(), "url", writeURL)
	}
	return err
}

// NewEdsAPI creates a new NewEdsAPI instance
//
//	address is optional to override the discovery
//	loginName if needed, "" if not needed
//	password if needed, "" if not needed
func NewEdsAPI(address string, loginName string, password string) *EdsAPI {
	edsAPI := &EdsAPI{
		address:         address,
		loginName:       loginName,
		password:        password,
		discoTimeoutSec: 3, // discovery timeout
	}
	return edsAPI
}
