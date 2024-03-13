package service

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/huin/goupnp"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// GET /rest/config
type ISYConfiguration struct {
	DeviceSpecs struct {
		Make      string `xml:"make"`            // Universal Devices Inc.
		Manuf     string `xml:"manufacturerURL"` // http://www.universal-devices.com
		Model     string `xml:"model"`           // Insteon Web Controller
		Icon      string `xml:"icon"`            // /web/udlogo.jpg
		Archive   string `xml:"archive"`         // /web/insteon.jar
		Chart     string `xml:"chart"`           // /web/chart.jar
		QueryInit bool   `xml:"queryOnInit"`     // true
		OneNode   bool   `xml:"oneNodeAtATime"`  // true
	} `xml:"deviceSpecs"`
	App            string `xml:"app"`             // Insteon_UD99
	AppVersion     string `xml:"app_version"`     // 3.2.6
	Platform       string `xml:"platform"`        // ISY-C-99
	BuildTimestamp string `xml:"build_timestamp"` // 2012-05-04-00:26:24

	Root struct {
		ID   string `xml:"nodeID"` // MAC  aa:bb:cc:dd:ee:ff
		Name string `xml:"name"`   // ISY gateway name customizable (might affect programs)
	} `xml:"root"`
	Product struct {
		ID          string `xml:"nodeID"` // 1020
		Description string `xml:"desc"`   // ISY 99i 256
	} `xml:"product"`
	//} `xml:"configuration"`
}

// IsyAPI manages the connection to the ISY99x gateway
// This supports publishing REST and Soap requests.
// Use of WebSockets is tbd
type IsyAPI struct {
	// address contains the IP address of the ISY gateway, or a path to a file.
	// if address starts with file:// then this is used to read from a
	// (simulation) file named <address>/<restPath>.xml
	address  string
	login    string // Basic Auth login name
	password string // Basic Auth password
	//simulation map[string]string // map used when in simulation

	isConnected atomic.Bool

	// the ISY connect gateway device config, if isConnected is true
	isyConfig ISYConfiguration

	// mutex to protect reconnection
	mux sync.RWMutex
}

// Connect to the gateway and read its ID
func (isyAPI *IsyAPI) Connect(
	gwAddr string, login string, password string) (err error) {

	if gwAddr == "" {
		gwAddr, err = isyAPI.Discover(time.Second * 3)
		if err != nil {
			isyAPI.isConnected.Store(false)
			return err
		}
	}
	isyAPI.mux.Lock()
	defer isyAPI.mux.Unlock()

	isyAPI.address = gwAddr
	isyAPI.login = login
	isyAPI.password = password

	// get the ISY gateway ID and test if it is reachable
	err = isyAPI.SendRequest("GET", "/rest/config", "", &isyAPI.isyConfig)
	if err != nil {
		// connection failed
		isyAPI.isConnected.Store(false)
	}
	isyAPI.isConnected.Store(true)

	// TODO: websocket connect
	return err
}

func (isyAPI *IsyAPI) Disconnect() {
	isyAPI.isConnected.Store(false)
	// TODO: websocket disconnect
}

// Discover any ISY99x gateway on the local network for 3 seconds.
// If successful this returns the ISY99x gateway address, otherwise an error is returned.
func (isyAPI *IsyAPI) Discover(timeout time.Duration) (addr string, err error) {
	slog.Info("Starting discovery")
	deviceType := "urn:udi-com:device:X_Insteon_Lighting_Device:1"
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	devices, err := goupnp.DiscoverDevicesCtx(ctx, deviceType /*ssdp.SSDPAll*/)
	cancelFn()

	if len(devices) == 0 {
		slog.Info("No ISY99x device found")
		return "", fmt.Errorf("no ISY99x device found")
	}
	// success!
	d0 := devices[0]
	slog.Info("Success! Found ISY99x device.",
		"addr", d0.Location.Host,
		"name", d0.Root.Device.FriendlyName,
		"manufacturer", d0.Root.Device.Manufacturer,
		"modelName", d0.Root.Device.ModelName,
		"modelNumber", d0.Root.Device.ModelNumber,
	)

	return d0.Location.Host, err
}

// GetNodesConfig returns info on the nodes
//func (isyAPI *IsyAPI) GetNodesConfig() ([]eds.XMLNode, error) {
//
//	soapTemplate := `<s:Envelope><s:Body>
//  <u:GetNodesConfig xmlns:u="urn:udi-com:service:X_Insteon_Lighting_Service:1">
//  </u:GetNodesConfig>
//</s:Body></s:Envelope>`
//
//	response := eds.XMLNode{}
//	err := isyAPI.SendRequest("POST", "/services", soapTemplate, &response)
//	if err != nil {
//		return nil, err
//	} else if len(response.Nodes) == 0 || len(response.Nodes[0].Nodes) == 0 {
//		return nil, fmt.Errorf("GetNodesConfig. Unexpected result")
//	}
//	nodesConfig := response.Nodes[0].Nodes[0]
//	return nodesConfig.Nodes, err
//}

// GetID returns the ISY Gateway ID
func (isyAPI *IsyAPI) GetID() string {
	// Use the 'app' field as the gateway ID
	return isyAPI.isyConfig.App
}

func (isyAPI *IsyAPI) IsConnected() bool {
	return isyAPI.isConnected.Load()
}

// ReadNodes reads the ISY Node list
func (isyAPI *IsyAPI) ReadNodes() (*IsyNodes, error) {
	isyNodes := IsyNodes{}
	err := isyAPI.SendRequest("GET", "/rest/nodes", "", &isyNodes)

	return &isyNodes, err
}

// Rename sets a new friendly name of the ISY device
func (isyAPI *IsyAPI) Rename(nodeID string, newName string) error {
	// Post a SOAP message as no REST api exists for this
	soapTemplate := `<s:Envelope><s:Body>
  <u:RenameNode xmlns:u="urn:udi-com:service:X_Insteon_Lighting_Service:1">
    <id>%s</id>
    <name>%s</name>
  </u:RenameNode>
</s:Body></s:Envelope>
`
	msgBody := fmt.Sprintf(soapTemplate, nodeID, newName)
	reply := make(map[string]interface{})
	isyAPI.mux.Lock()
	defer isyAPI.mux.Unlock()
	err := isyAPI.SendRequest("POST", "/services", msgBody, &reply)
	return err
}

// SendRequest sends a GET or PUT request to the ISY device
//
//		method is the GET or POST method to use.
//		restPath contains REST or /services SOAP path to send the request to.
//	 body optional body for post
//		If it starts with file:// then read from (simulation) file named <address>/<restPath>.xml
func (isyAPI *IsyAPI) SendRequest(method string, restPath string, body string, response interface{}) error {
	// if address is a file then load content from file. Intended for testing
	if strings.HasPrefix(isyAPI.address, "file://") {
		filename := path.Join(isyAPI.address[7:], restPath+".xml")
		buffer, err := os.ReadFile(filename)
		if err != nil {
			slog.Error("isyRequest: Unable to read ISY data from file", "filename", filename, "err", err)
			return err
		}
		err = xml.Unmarshal(buffer, &response)
		return err
	}

	// not a file, continue with http request
	isyURL := "http://" + isyAPI.address + restPath
	var reqBody io.Reader
	if body != "" {
		reqBody = bytes.NewReader([]byte(body))
	}
	req, err := http.NewRequest(method, isyURL, reqBody)

	if err != nil {
		return err
	}
	req.SetBasicAuth(isyAPI.login, isyAPI.password)
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		slog.Error("isyRequest: Unable to read ISY device", "URL", isyURL, "err", err)
		return err
	} else if resp.StatusCode != 200 {
		msg := fmt.Sprintf("isyRequest: Error code return by ISY device %s: %v", isyURL, resp.Status)
		slog.Warn(msg)
		err = errors.New(msg)
		return err
	}

	// Decode the response into XML
	if response != nil {
		data, err := io.ReadAll(resp.Body)
		_ = data
		_ = err
		//dec := xml.NewDecoder(resp.Body)
		//err = dec.Decode(&response)
		err = xml.Unmarshal(data, response)
		//err = dec.Decode(response)
		_ = resp.Body.Close()
	}

	return err
}

// NewIsyAPI create an ISY connection handler instance.
//
//	gwAddress is the ip address of the gateway, or "file://<path>" to a simulation xml file
//	login to gateway device
//	password to gateway device
func NewIsyAPI() *IsyAPI {
	isy := &IsyAPI{}
	//isy.simulation = make(map[string]string)
	return isy
}
